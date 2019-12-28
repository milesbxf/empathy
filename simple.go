package empathy

import (
	"flag"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

var (
	persistentMode bool
	kindCluster    *KindCluster
	kubeClient     kubernetes.Interface
)

func init() {
	flag.BoolVar(&persistentMode, "persistent-kind-cluster", false, "Whether to use persistent test K8s clusters between runs")
}

func Init(clusterName string) error {
	return InitWithConfig(
		clusterName,
		`
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
    `)
}

func InitWithConfig(clusterName, kindConfig string) error {
	if !flag.Parsed() {
		flag.Parse()
	}

	if kindCluster != nil {
		panic("empathy already initialized")
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "getting user home dir")
	}

	empathyBaseDir := path.Join(userHomeDir, ".kube", "empathy")
	if err := os.MkdirAll(empathyBaseDir, 0700); err != nil {
		return errors.Wrap(err, "creating empathy base dir")
	}

	kubeconfigPath := path.Join(empathyBaseDir, clusterName)

	kubeconfig, err := os.OpenFile(kubeconfigPath, os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return errors.Wrap(err, "open/create kubeconfig")
	}

	kindCluster, err = Create(
		clusterName,
		kubeconfig.Name(),
		`
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
    `)
	return err
}

func TearDown() error {
	if !persistentMode {
		if err := Cluster().Delete(); err != nil {
			return err
		}
		if err := os.Remove(Cluster().Kubeconfig()); err != nil {
			return err
		}
	}

	return nil
}

func Cluster() *KindCluster {
	if kindCluster == nil {
		panic("empathy not initilized; call one of the Init* methods first!")
	}
	return kindCluster
}

func Clientset() kubernetes.Interface {
	if kindCluster == nil {
		panic("empathy not initilized; call one of the Init* methods first!")
	}
	if kubeClient == nil {
		kubeClient = kubernetes.NewForConfigOrDie(Cluster().RESTConfig())
	}
	return kubeClient
}

func InstallManifestGlob(glob string) error {
	if kindCluster == nil {
		panic("empathy not initilized; call one of the Init* methods first!")
	}

	manifests, err := filepath.Glob(glob)
	if err != nil {
		return err
	}
	return Cluster().InstallManifests(manifests)
}

package empathy

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/pkg/errors"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/kind/pkg/cluster"
	kindcmd "sigs.k8s.io/kind/pkg/cmd"
)

// KindCluster represents a kind Kubernetes cluster, with kubeconfig (for external clients like kubectl) and rest.Config for in-process clients
type KindCluster struct {
	Name       string
	kubeconfig *os.File
	config     *rest.Config
}

// Creates a cluster, storing kubeconfig at the given path
// If a cluster already exists, we return a KindCluster object pointing to it
// Allows passing in YAML config to configure kind, e.g. to expose ports or add more nodes
// See https://github.com/kubernetes-sigs/kind/blob/master/site/content/docs/user/quick-start.md#configuring-your-kind-cluster
func Create(name, kubeconfigPath, kindConfig string) (*KindCluster, error) {
	provider := cluster.NewProvider(cluster.ProviderWithLogger(kindcmd.NewLogger()))

	c := &KindCluster{
		Name: name,
	}
	if f, err := os.Open(kubeconfigPath); os.IsNotExist(err) {
		c.kubeconfig, err = os.Create(kubeconfigPath)
		if err != nil {
			return nil, err
		}
	} else {
		c.kubeconfig = f
	}

	n, err := provider.ListNodes(name)
	if err != nil {
		return nil, err
	}
	if len(n) == 0 {
		if err := kindCreate(name, kubeconfigPath, kindConfig); err != nil {
			return nil, err
		}
	}

	fmt.Printf("💥 Cluster %s ready. You can access it by setting:\nexport KUBECONFIG='%s'\n", name, kubeconfigPath)
	return c, nil
}

// Delete removes the cluster from kind. The cluster may not be deleted straight away - this only issues a delete command
func (c *KindCluster) Delete() error {
	provider := cluster.NewProvider(cluster.ProviderWithLogger(kindcmd.NewLogger()))
	return provider.Delete(c.Name, c.kubeconfig.Name())
}

// Kubeconfig returns the path to the cluster kubeconfig
func (c *KindCluster) Kubeconfig() string {
	return c.kubeconfig.Name()
}

// RESTConfig returns K8s client config to pass to clientset objects
func (c *KindCluster) RESTConfig() *rest.Config {
	if c.config == nil {
		var err error
		c.config, err = clientcmd.BuildConfigFromFlags("", c.Kubeconfig())
		if err != nil {
			panic(err)
		}
	}
	return c.config
}

// InstallManifests applies manifests from the given fileglob using kubectl
func (c *KindCluster) InstallManifests(manifests []string) error {
	for _, m := range manifests {
		fmt.Printf("ℹ️  Installing manifest %s....\n", m)

		// Would much rather do this programmatically in Go rather than using os/exec, but it's very nontrivial to handle arbitrary manifest types (e.g. CRDs)
		cmd := exec.Command("kubectl", "apply", "-f", m, "--kubeconfig", c.Kubeconfig())
		stdoutStderr, err := cmd.CombinedOutput()
		if err != nil {
			return errors.Wrapf(err, "failed to apply manifest %s. Output:\n%s", m, stdoutStderr)
		}
	}
	return nil
}

// kindCreate creates the kind cluster. It will retry up to 10 times if cluster creation fails.
func kindCreate(name, kubeconfig, kindConfig string) error {

	fmt.Printf("🌧️  Creating kind cluster %s...\n", name)
	provider := cluster.NewProvider(cluster.ProviderWithLogger(kindcmd.NewLogger()))
	attempts := 0
	maxAttempts := 10
	for {
		err := provider.Create(
			name,
			cluster.CreateWithNodeImage(""),
			cluster.CreateWithRetain(false),
			cluster.CreateWithWaitForReady(time.Duration(0)),
			cluster.CreateWithKubeconfigPath(kubeconfig),
			cluster.CreateWithDisplayUsage(false),
			cluster.CreateWithRawConfig([]byte(kindConfig)),
		)
		if err == nil {
			return nil
		}

		fmt.Printf("Error bringing up cluster, will retry (attempt %d): %v", attempts, err)
		attempts++
		if attempts >= maxAttempts {
			return errors.Wrapf(err, "Error bringing up cluster, exceeded max attempts (%d)", attempts)
		}
	}
}

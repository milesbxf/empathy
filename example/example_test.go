package example

import (
	"io/ioutil"
	"path"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/milesbxf/empathy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tjarratt/babble"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func assertBusyboxPod(t *testing.T, kubeclient kubernetes.Interface) {
	if err := wait.PollImmediate(
		100*time.Millisecond,
		30*time.Second,
		func() (bool, error) {

			podList, err := kubeclient.CoreV1().Pods("default").List(metav1.ListOptions{
				LabelSelector: "app=busybox1",
			})
			if err != nil {
				return false, nil
			}

			if len(podList.Items) == 0 {
				return false, nil
			}

			if podList.Items[0].Spec.Containers[0].Name == "busybox" {
				return true, nil
			}

			return false, nil
		},
	); err != nil {
		assert.Fail(t, "Timed out waiting for pod with busybox container")
	}
}

func TestBatteriesIncluded(t *testing.T) {
	empathy.Init("empathy-examples-test-batteries-included")
	defer empathy.TearDown()

	_, currentFile, _, _ := runtime.Caller(0)
	manifestPath := path.Join(path.Dir(currentFile), "manifests", "*.yaml")

	err := empathy.InstallManifestGlob(manifestPath)
	require.NoError(t, err)

	assertBusyboxPod(t, empathy.Clientset())
}

func TestAdvancedClusterUsage(t *testing.T) {
	babbler := babble.NewBabbler()

	kubeconfig, err := ioutil.TempFile("", "kubeconfig")
	require.NoError(t, err)
	defer kubeconfig.Close()

	cluster, err := empathy.Create(
		babbler.Babble(),
		kubeconfig.Name(),
		`
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
    `)
	require.NoError(t, err)

	defer func() {
		err = cluster.Delete()
		require.NoError(t, err)
	}()

	_, currentFile, _, _ := runtime.Caller(0)
	manifestPath := path.Join(path.Dir(currentFile), "manifests", "*.yaml")

	manifests, err := filepath.Glob(manifestPath)
	require.NoError(t, err)

	err = cluster.InstallManifests(manifests)
	require.NoError(t, err)

	restConfig := cluster.RESTConfig()
	kubeclient := kubernetes.NewForConfigOrDie(restConfig)

	assertBusyboxPod(t, kubeclient)
}

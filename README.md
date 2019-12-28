# Empathy

empathy allows you to easily create Kubernetes clusters from Go code using [kind](https://github.com/kubernetes-sigs/kind/).

```
empathy.Init("my-test")
defer empathy.TearDown()

empathy.InstallManifestGlob("manifests/*.yaml")

kubeClient := empathy.Clientset()
```

Whilst kind itself works well for adhoc local development and end-to-end tests orchestrated by Bash/Make, empathy brings it to the use case of writing end-to-end/integration tests directly in Go.


## Usage

To create a cluster with default config, just call `empathy.Init("<name>")`, and make sure you call `empathy.TearDown()` to clean up afterwards. If you want to reuse the same cluster between tests, put these in your [`TestMain`](https://golang.org/pkg/testing/#hdr-Main) test setup function.

You can also call `empathy.InitWithConfig("<name>", kindConfig)` to specify custom kind config (in YAML), if you want to do things like add additional nodes or map ports from the host. See here: https://github.com/kubernetes-sigs/kind/blob/master/site/content/docs/user/quick-start.md#configuring-your-kind-cluster

You can get a default Kubernetes clientset with `empathy.Clientset()`. If you'd like to initialize your own client (e.g. for custom types), you can access the raw REST config via `empathy.Cluster().RESTConfig()`.


### Applying manifests

You'll likely want to provision some resources in the new cluster. Whilst you could do this programmatically with the Kubernetes clientset, that can be pretty tedious. Empathy includes a helper method to run `kubectl apply` against all manifests matching a certain fileglob:

```
empathy.InstallManifestGlob("manifests/*.yaml")
```

If you simply have a list of known manifests you want to apply, try:

```
empathy.Cluster().InstallManifests([]string{"deploy.yaml", "svc.yaml"})
```


### Keeping persistent clusters

By default, empathy creates ephemeral clusters which are deleted when you run `empathy.TearDown()`. This works great for running tests against a known clean state, but is quite slow. It's also much easier to debug test failures if you can directly access the cluster they're running against.

Empathy adds a `-persistent-kind-cluster` test flag which skips the delete phase. The next time you run the tests, empathy will automatically reuse the cluster with the same name. In the test output, look out for a line pointing to the `KUBECONFIG` path - if you run that line, you can use your local `kubectl` to access the cluster.


### Advanced usage

`empathy.Init("<name>")` only works with a single cluster - if you call `Init` multiple times it will fail. There are some use cases where you might want to create multiple clusters - in this case, you can use `empathy.Create`, which returns a reference to each cluster which you'll need to keep track of yourself:
```
cluster1, err := empathy.Create("cluster1", "~/.kube/cluster1-config", `
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
`)

cluster2, err := empathy.Create("cluster1", "~/.kube/cluster2-config", `
kind: Cluster
apiVersion: kind.sigs.k8s.io/v1alpha3
`)

clientset1 := kubernetes.NewForConfigOrDie(cluster1.RESTConfig())
clientset2 := kubernetes.NewForConfigOrDie(cluster2.RESTConfig())

clientset1.CoreV1().Pods("default").List(metav1.ListOptions{})
...
```


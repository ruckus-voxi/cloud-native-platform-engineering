# Automatically Obtain Default Login Creds

From the perspective of `aplcli` (client side), the deployment
process is complete once then Helm chart installation (server side)
has taken over. This is to maintain a clear separation of concerns.
The CLI will not monitor the Helm process for you. It does however, write a copy of your cluster's `kubeconfig` to
`$HOME/.kube/<name>-kubeconfig.yaml`. Replace `<name>` with the platform `name` defined in your `aplcli`
config. See [Initialize config](#initialize-config) for example. 

With your `kubeconfig` file, you can monitor the Helm process by
following the logs from the `apl` job in the default namespace.

```bash
export KUBECONFIG="$HOME/.kube/apl-ams-kubeconfig.yaml"
kubectl logs job/apl -f
```

The Helm installation takea approximately 5-10 minutes. A banner
displaying a succcess message denotes that the process is complete,
and your platform is now ready for initial login. It also displays
the console URL to visit in your browser, along with the two
`kubectl` commands to retreive the initial login username and
password. Keycloak enforces updating this password on the first
login.

```bash
# platform admin username
kubectl get secret platform-admin-initial-credentials -n keycloak -o jsonpath='{.data.username}' | base64 -d

# initial password
kubectl get secret platform-admin-initial-credentials -n keycloak -o jsonpath='{.data.password}' | base64 -d
```

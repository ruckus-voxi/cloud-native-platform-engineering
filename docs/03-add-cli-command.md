# Add a CLI Command

APL CLI becomes much more useful if we can also use it for post-installation CRUD operations. Fortunately the App Platform is API driven, which enables us to do just that. A client library is already provided and available for importing into CLI commands―no further action is required it start using it. This [doc](api/client-codegen.md) outlines the process used to generate it.

APL CLI was built using the popular `cobra-cli` [generator tool](https://github.com/spf13/cobra/), and `viper` [configuration solution](https://github.com/spf13/viper). Ensure that both of these are installed on your system before proceeding further.

```bash
go install github.com/spf13/cobra-cli@latest
go get -u github.com/spf13/viper
go mod tidy
```

Let's add a `team` command that utilizes this client library to manage teams on an APL instance. From the repository root, run the below `cobra-cli` command to generate `team.go` with some boilerplate code.

```bash
cobra-cli add team
```

You should see it appear under the `cmd` directory.

```bash
tree cmd/ -L 1
cmd/
├── automation.go
├── cleanup.go
├── create.go
├── deploy.go
├── destroy.go
├── esc.go
├── gen.go
├── help.go
├── logger.go
├── rclone.go
├── root.go
├── setup.go
├── team.go  <--
└── templates
```

Then to make this lab a little easier, use the kubeconfig file from one of your APL Kubernetes clusters and set some environment variables in your shell. If the name of your cluster/APL instance is `my-platform` for example, set that corresponding kubeconfig file, and then use it with `kubectl` to query `keycloak` for the realm username and password.

```bash
export KUBECONFIG=$HOME/.kube/my-platform-kubeconfig.yaml

# keycloak credentials for obtaining api tokens
export CLIENT_SECRET=$(kubectl get -n apl-keycloak-operator secrets apl-keycloak-operator-secret -ojson | jq -r '.data.KEYCLOAK_CLIENT_SECRET' | base64 -d)
export USER_NAME=$(kubectl get secret keycloak-initial-admin -n keycloak -o json | jq -r '.data.username' | base64 -d)
export USER_PASSWORD=$(kubectl get secret keycloak-initial-admin -n keycloak -o json | jq -r '.data.password' | base64 -d)
```
____
## Lab Challenge

Begin writing the `team` command logic in this `team.go` source file. Also open up the `client.go` file we generated earlier, to see what you need to import. Ensure that it can minimally satisfy the following requirements, for any existing APL instance named from your `aplcli` configuration file:

- Add new teams
- Remove existing teams
- List existing teams

> [!TIP]
> Use the same format of a `--name/-n` flag that other commands use, in order to specify the APL instance to act on. For that matter, you'll likely save a ton of time by reviewing how those other commands work.

You can also look at the [example solution](./solutions/03-add-cli-command.md)if you're feeling stuck somewhere, but try your best to finish the lab without taking a peek. Also consider that the example solution is not perfect, nor is it the only way to factor your code for achieving this task. Good luck!

> [!TIP]
> By default the API token generated via the `keycloak` client secret has a short expiry. Your code needs to account for this. To get an idea for the process involved in getting an access token, run these commands separately in your terminal.
> ```bash
> export TOKEN=$(curl -s -X POST "https://keycloak.${DOMAIN}/realms/otomi/protocol/openid-connect/token" -H "Content-Type: application/x-www-form-urlencoded" -d "client_id=otomi" -d "client_secret=${CLIENT_SECRET}" -d "grant_type=password" -d "username=${USER_NAME}" -d "password=${USER_PASSWORD}" | jq -r .access_token)
> ```

### Bonus

In the above lab instructions, you manually set the `$KUBECONFIG`, `$CLIENT_SECRET`, `$USER_NAME`, and `$USER_PASSWORD` environment variables in your shell. Update the `team` command to do this programmatically.

> [!TIP]
> The correct file for `$KUBECONFIG` can be matched by the value provided to the `--name/-n` flag.
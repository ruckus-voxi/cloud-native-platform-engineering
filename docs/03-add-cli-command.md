# Generate App Platform API Client Library

The App Platform is API driven. It adhears to the [OpenAPI 3.0 specification](https://swagger.io/specification/v3/) and exposes an endpoint for users to access the Swagger web UI for documentation and testing―you've probably seen [these](https://petstore.swagger.io/?_gl=1*87igsc*_gcl_au*MTQzMDc3NTkwNS4xNzcxMDMxMDQ1) before. This standardization enables the use of tooling which can consume the JSON schema and auto-generate boilerplate code for us. In the Golang arena, One such tool is the remarkable [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen). We can use it to generate HTTP models, server-side code, as well as clients. The latter is what we'll focus on for this exercise.

Let's get started by adding this tool to our arsenal. For systems running Go 1.24+, the offical project recommends installing it `go get -tool`, but the `go install` works as well. We won't dig into the reasons for picking one or the other, but if you're interested, a co-maintainer of the `oapi-codegen` project outlines it pretty well this [post](https://www.jvt.me/posts/2025/01/27/go-tools-124/).  

```bash
go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

# alternatively use `go install`
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

Next, if you haven't already, login to your APL console dashboard at `https://console.${DOMAIN}`. Be sure to repalce `${DOMAIN}` with the actual domain/subdomain of your APL installation. Once logged in you can visit the Swagger UI at `https://console.${DOMAIN}/api/api-docs/swagger/`. Look around for bit and get familiar, you'll notice there are both `/v1` and `/v2` endpoints. The differences are not immediately clear, but for a short example, you'll find API definitions for _team-specific_ configuration from underneath of `/v1/teams/{teamid}` (images, workloads, etc) or _platform specific_ settings from `/v2` or `/v1`. With some rare exceptions (when the API is not finalized or there are new apps under testing), the entire values structure is represented in these API definitions.

Remove `/swagger/` from the URL path and reload the page, so that you are visiting `https://console.${DOMAIN}/api/api-docs/`. You should see the massive blob of JSON representing every possible API endpoint. Highlight all of it and copy/paste to a file called `api.yaml`.

Next you'll create an `oapi-codegen` config file called `cfg.yaml`. The values for `package` and `output` can be customized if you'd like, but for the purpose of following along to setup for this exercise, you may find it easier to just copy the example below.

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/oapi-codegen/oapi-codegen/HEAD/configuration-schema.json
package: client
output: client.go
generate:
  models: true
  client: true
```

Now the start of the fun part―run the below command and watch your new client library appear in a file called `client.go`. The part we are most interested in starts at around line `4442` with the `Client` struct.

```bash
oapi-codegen -config cfg.yaml api.yaml
```

For reference, the `aplcli` directory tree looks like this after a fresh `git clone` and `cd` into it.

```bash
.
├── cmd
│   └── templates
│       ├── apl
│       ├── ci
│       ├── infra
│       ├── init
│       ├── pulumi
│       ├── utils
│       └── values
├── config
├── docs
│   └── solutions
└── media
    ├── images
    └── screencasts
```

Create a new `client` subdirectory under `cmd` and then move or copy the newley generated `client.go` file to it. Assuming you currently in the repo root, and you ran the previous `oapi-codgen` command from one directory outside of it, the two commands you need to run are:

```bash
mkdir cmd/client
cp ../client.go cmd/client/
```

Now before we start importing this module into our CLI code, we need to quickly install a few more dependencies, the first being another package by the `oapi-codegen` project called `securityprovider`. This provides a `RequestEditorFn` type our client uses to intercept and mutate HTTP requests―we use it for [Bearer Authentication](https://swagger.io/docs/specification/v3_0/authentication/bearer-authentication/) in this case. See the [Authenticated API Example](https://github.com/oapi-codegen/oapi-codegen/blob/main/examples/authenticated-api/README.md) for additional info. 

```bash
go get github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider
```

Then be sure you have the `cobra-cli` [generator tool](https://github.com/spf13/cobra/) installed on your system, and the `viper` [configuration solution](https://github.com/spf13/viper) installed as a module.

```bash
go install github.com/spf13/cobra-cli@latest
go get -u github.com/spf13/viper
go mod tidy
```

We are finally at the fun part...let's write some code! Let's use `cobra-cli` to get us started with adding a `team` command to our own CLI, which enables adding, removing, and listing of developer teams of an App Platform instance. From the root of repository, run the following to generate `team.go`. You should see it appear under the `cmd` directory.

```bash
cobra-cli add team
```
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

Then to make this lab a little easier, use the kubeconfig file from one of your APL Kubernetes clusters and set some environment variables. If the name of your cluster/APL instance is for example, `my-platform`, set that corresponding kubeconfig file, and then use it with `kubectl` to query `keycloak` for the realm username and password.

```bash
export KUBECONFIG=$HOME/.kube/my-platform-kubeconfig.yaml
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
> Use the same format of a `--name/-n` flag that ohter commands use, in order to specify the APL instance to act on. For that matter, you'll likely save a ton of time by reviewing how those other commands work.

You can also look at the [example solution](./solutions/03-add-cli-command.md)if you're feeling stuck somewhere, but try your best to finish the lab without taking a peek. Also consider that the example solution is not perfect, nor is it the only way to factor your code for achieving this task. Good luck!

> [!TIP]
> By default the API token generated via the `keycloak` client secret has a short expiry. Your code needs to account for this. To get an idea for the process involved in getting an access token, run these commands separately in your terminal.
> ```bash
> export TOKEN=$(curl -s -X POST "https://keycloak.ams.arch-linux.io/realms/otomi/protocol/openid-connect/token" -H "Content-Type: application/x-www-form-urlencoded" -d "client_id=otomi" -d "client_secret=${CLIENT_SECRET}" -d "grant_type=password" -d "username=${USER_NAME}" -d "password=${USER_PASSWORD}" | jq -r .access_token)
> ```

### Bonus

If you nailed the challenge of adding the `team` command, and thirsty for more, keep going! Add another command called something like `user` that just like the `team` command, imports generated code from `client.go`. Also ensure it can target a specific APL istance, and adds, removes, and lists users on it.
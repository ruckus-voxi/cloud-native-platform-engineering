# Add CLI Command: Solution

```go
package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	oapi "github.com/akamai-developers/aplcli/client"
	"github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider"
	"github.com/spf13/cobra"
)

var (
	apiURL    string
	teamName  string
	addTeam   bool
	rmTeam    bool
	listTeams bool
	teamId    string
)

// teamCmd represents the team command
var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Add or Remove teams to APL",
	PreRun: func(cmd *cobra.Command, args []string) {
		apiURL = "https://api." + platform.Domain

		// make --add and --remove mutually exlusive
		errMsg := "parse flag values: "
		if addTeam && rmTeam {
			cmd.Help()
			logger.Error(errMsg + "--add and --remove are mutually exclusive")
		}

		// require --team-id togehter with other flags except for --list
		if !listTeams && teamId == "" {
			cmd.Help()
			logger.Error(errMsg + "--team-id [string] is required")
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		var ()

		ctx := context.Background()
		client := aplClient(ctx, apiURL)

		switch {
		case listTeams:
			var jsonDataArray []map[string]any

			res, err := client.GetTeams(ctx)
			handleErr("list teams", err)
			chkStatusCode(res.StatusCode)
			jsonPrettyPrint(res.Body, jsonDataArray)
		case addTeam:
			var jsonData map[string]any

			teamIdPtr := func(string) *string { return &teamId }(teamId)
			jsonReq := oapi.CreateTeamJSONRequestBody{
				Id:   teamIdPtr,
				Name: teamId,
			}

			res, err := client.CreateTeam(ctx, jsonReq)
			handleErr("add new team", err)
			chkStatusCode(res.StatusCode)
			jsonPrettyPrint(res.Body, jsonData)
		case rmTeam:
			res, err := client.DeleteTeam(ctx, teamId)
			handleErr("remove team", err)
			chkStatusCode(res.StatusCode)
			logger.Info("deleted team: " + teamId)
		}
	},
}

func init() {
	rootCmd.AddCommand(teamCmd)

	teamCmd.Flags().StringVarP(&teamName, "name", "n", "", "APL instance name (required)")
	teamCmd.MarkFlagRequired("name") // makes the --name flag required
	teamCmd.Flags().BoolVarP(&addTeam, "add", "a", false, "Create a new team")
	teamCmd.Flags().BoolVarP(&rmTeam, "remove", "r", false, "Remove an existing team")
	teamCmd.Flags().StringVarP(&teamId, "team-id", "", "", "Team name (required with --add/--remove)")
	teamCmd.Flags().BoolVarP(&listTeams, "list", "l", false, "List all teams in APL instance")
}

// aplClient is a wrapper func around the oapi-codegen generated
// client.NewClient method, the securityprovider.NewSecurityProviderBearerToken
// method, and handling of the http request. It returns an authenticated client.
func aplClient(ctx context.Context, apiURL string) *oapi.Client {
	endpoint := strings.NewReplacer("api.", "keycloak.").Replace(apiURL)
	endpoint += "/realms/otomi/protocol/openid-connect/token"

	formData := map[string]string{
		"client_id":     "otomi",
		"client_secret": os.Getenv("CLIENT_SECRET"),
		"grant_type":    "password",
		"username":      os.Getenv("USER_NAME"),
		"password":      os.Getenv("USER_PASSWORD"),
	}

	formValues := url.Values{}

	for k, v := range formData {
		formValues.Set(k, v)
	}

	payload := bytes.NewBufferString(formValues.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, payload)
	if err != nil {
		logger.Error("create http post request for api token from keycloak: " + err.Error())
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	httpClient := &http.Client{}
	res, err := httpClient.Do(req)
	if err != nil {
		handleErr("make http post request for api token from keycloak", err)
	}

	defer res.Body.Close()

	var (
		resData map[string]any
		token   string
	)

	if err := json.NewDecoder(res.Body).Decode(&resData); err != nil {
		handleErr("decode http response body", err)
	}

	if t, ok := resData["access_token"].(string); ok {
		token = t
	} else {
		err := fmt.Errorf("invalid response data")
		handleErr("parse JSON response body for access token", err)
	}

	bearer, err := securityprovider.NewSecurityProviderBearerToken(token)
	if err != nil {
		handleErr("create bearer token security provider", err)
	}

	c, err := oapi.NewClient(apiURL, oapi.WithRequestEditorFn(bearer.Intercept))
	if err != nil {
		handleErr("create apl api client", err)
	}

	return c
}

func handleErr(msg string, err error) {
	if err != nil {
		logger.Error(msg + ": " + err.Error())
		os.Exit(1)
	}
}

func chkStatusCode(status int) {
	if status != http.StatusOK {
		err := fmt.Errorf("status code %s", strconv.Itoa(status))
		handleErr("verify the request/response status code", err)
	}
}

func jsonPrettyPrint(response io.ReadCloser, data any) {
	err := json.NewDecoder(response).Decode(&data)
	handleErr("decode json data", err)

	out, err := json.MarshalIndent(data, "", "  ")
	handleErr("indent json output for pretty print", err)
	logger.Info(string(out))
}
```

The App Platform is API driven. It adhears to the [OpenAPI 3.0 specification](https://swagger.io/specification/v3/) and exposes an endpoint for users to access the Swagger web UI for documentation and testing―you've probably seen [these](https://petstore.swagger.io/?_gl=1*87igsc*_gcl_au*MTQzMDc3NTkwNS4xNzcxMDMxMDQ1) before. This standardization enables the use of tooling which can consume the JSON schema and auto-generate boilerplate code for us. In the Golang arena, One such tool is the remarkable [oapi-codegen]. We can use it to generate HTTP models, server-side code, as well as clients. The latter is what we'll focus on for this exercise.

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

____
## Lab Challenge

Begin writing the `team` command logic in this `team.go` source file. Also open up the `client.go` file we generated earlier, to see what you need to import. Ensure that it can minimally satisfy the following requirements, for any existing APL instance named from your `aplcli` configuration file:

- Add new teams
- Remove existing teams
- List existing teams

> [!TIP]
> Use the same format of a `--name/-n` flag that ohter commands use, in order to specify the APL instance to act on. For that matter, you'll likely save a ton of time by reviewing how those other commands work. You can also look at the [example solution](./solutions/client.go) if you're feeling stuck somewhere, but try your best to finish the lab without taking a peek. Also consider that the example solution is not perfect, nor is it the only way to factor your code for achieving this task. Good luck!


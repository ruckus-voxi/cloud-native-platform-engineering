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

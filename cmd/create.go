package cmd

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Platform struct {
	Email       string   `yaml:"email,omitempty"`
	Domain      string   `yaml:"domain,omitempty"`
	AplVersion  string   `yaml:"aplversion,omitempty"`
	KubeVersion string   `yaml:"kubeversion,omitempty"`
	Name        string   `yaml:"name,omitempty"`
	NbTag       string   `yaml:"nbtag,omitempty"`
	NodeCount   int      `yaml:"nodecount,omitempty"`
	NodeMax     int      `yaml:"nodemax,omitempty"`
	NodeType    string   `yaml:"nodetype,omitempty"`
	ObjPrefix   string   `yaml:"objprefix,omitempty"`
	Region      string   `yaml:"region,omitempty"`
	Repo        string   `yaml:"repo,omitempty"`
	Stack       string   `yaml:"stack,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Values      string   `yaml:"values,omitempty"`
}

var platform Platform

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create and bootstrap App Platform projects",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()

		projPath := filepath.Join(paths.Projects, platform.Name)
		values := filepath.Join(paths.Values, platform.Values)
		proj := NewProject(platform, projPath, values)

		// codegen
		proj.CodeGen()
		proj.Init(ctx)

		// initialize pulumi esc environment and generate admin passwords
		secMap := map[string]any{
			"developTeamPass": FnSecret(Passgen()),
			"lokiAdminPass":   FnSecret(Passgen()),
			"otomiAdminPass":  FnSecret(Passgen()),
		}

		// generate age provider sops keys
		ageKeys, err := GenAgeKeys()
		if err != nil {
			logger.Error("generate age keys: " + err.Error())
		}
		ageKeyMap := map[string]any{
			"publicKey":  ageKeys.Recipient().String(),
			"privateKey": FnSecret(ageKeys.String()),
		}

		token := map[string]any{
			"token": FnSecret(os.Getenv("LINODE_TOKEN")),
		}

		// ordered map of esc values to write
		escItems := map[int]EscEnvItem{
			1: {Name: "linode", Value: token},
			2: {Name: "age", Value: ageKeyMap},
			3: {Name: "aplSecrets", Value: secMap},
		}

		// pulumiConfig: https://tinyurl.com/pulumiconfig-esc
		escConfig := map[string]any{
			"linode:token": `${linode.token}`,
			"apl:age":      `${age}`,
			"apl:secrets":  `${aplSecrets}`,
		}

		esc := EscEnv{
			EnvName:  platform.Stack,
			Config:   escConfig,
			Items:    escItems,
			OrgName:  viper.GetString("pulumiOrg"),
			ProjName: platform.Name,
		}
		esc.Init()
	},
}

func init() {
	defaultTags := []string{
		"apl",
		"dev",
	}
	defaultRepo := "github.com/akamai-developers/aplcli"

	// required local flags
	createCmd.Flags().StringVarP(&platform.Name, "name", "n", "", "APL instance name (required)")
	createCmd.MarkFlagRequired("name") //nolint:errcheck
	createCmd.Flags().StringVarP(&platform.Domain, "domain", "d", "", "Domain or subdomain (required)")
	createCmd.Flags().StringVarP(&platform.Email, "email", "e", "", "SOA and cert-manager email (required)")
	createCmd.Flags().StringVarP(&platform.Region, "region", "r", "", "Akamai cloud region (required)")

	// optional local flags
	createCmd.Flags().StringVarP(&platform.AplVersion, "apl-version", "", "4.12.1", "App Platform version")
	createCmd.Flags().StringVarP(&platform.KubeVersion, "kube-version", "", "1.33", "Kubernetes version")
	createCmd.Flags().StringVarP(&platform.ObjPrefix, "obj-prefix", "", "apl", "S3 bucket label prefix")
	createCmd.Flags().StringVarP(&platform.NbTag, "nb-tag", "", "apl-static-lb", "NodeBalancer tag")
	createCmd.Flags().IntVarP(&platform.NodeCount, "node-count", "", 3, "Node pool count")
	createCmd.Flags().IntVarP(&platform.NodeMax, "node-max", "", 15, "Node pool autoscale max")
	createCmd.Flags().StringVarP(&platform.NodeType, "node-type", "", "g6-dedicated-8", "Node pool instance type")
	createCmd.Flags().StringVarP(&platform.Stack, "stack", "", "dev", "Pulumi stack name")
	createCmd.Flags().StringArrayVarP(&platform.Tags, "tags", "", defaultTags, "Cloud infra tags")
	createCmd.Flags().StringVarP(&platform.Repo, "repo", "", defaultRepo, "Repo URL")
	createCmd.Flags().StringVarP(&platform.Values, "values", "", valuesFile, "Helm chart values.yaml template")

	viper.BindPFlags(createCmd.LocalFlags()) //nolint:errcheck
}

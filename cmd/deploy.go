package cmd

import (
	"context"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deployTarget string

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy an App Platform project",
	PreRun: func(cmd *cobra.Command, args []string) {
		org := viper.GetString("PulumiOrg")
		ok := EscExists(org, platform.Name, platform.Stack)

		if !ok {
			logger.Error("esc environment not found: run 'create' command first")
			os.Exit(0)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		stacks := map[int]*MicroStack{
			1: {Name: "infra", PostRun: []string{"addNodeBalancerId"}},
			2: {Name: "apl"},
		}

		var idx int
		for k, i := range stacks {
			i.Path = filepath.Join(paths.Projects, platform.Name, "cmd", i.Name)
			i.GetFullName(ctx)

			if deployTarget == i.Name {
				idx = k
			}
		}

		switch {
		case idx > 0:
			if st, ok := stacks[idx]; ok {
				st.Up(ctx)
			}
		default:
			for i := 1; i < 3; i++ {
				if st, ok := stacks[i]; ok {
					st.Up(ctx)
				}
			}
		}
	},
}

func init() {
	// required flags
	deployCmd.Flags().StringVarP(&platform.Name, "name", "n", "", "APL instance name (required)")
	deployCmd.MarkFlagRequired("name") //nolint:errcheck
	// optional flags
	deployCmd.Flags().StringVarP(&deployTarget, "target", "t", "", "Target a specific project")

	viper.BindPFlags(deployCmd.LocalFlags()) //nolint:errcheck
}

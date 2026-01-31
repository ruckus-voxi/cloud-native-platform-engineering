package cmd

import (
	"context"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	destroyStacks StackMap
	destroyTarget string
	purgeAll      bool
	purgeEsc      bool
	purgeObj      bool
	purgeStk      bool
)

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Destroy existing App Platform projects and resources",
	PreRun: func(cmd *cobra.Command, args []string) {
		destroyStacks = StackMap{
			1: {Name: "apl", PostRun: []string{"cleanupLke"}},
			2: {Name: "infra", PostRun: []string{"rmNodeBalancerId"}},
		}

		addPrePostRun := func(action, s string, b bool, i int) {
			funcs := make([]string, 0)
			switch action {
			case "pre":
				funcs = destroyStacks[i].PreRun
			case "post":
				funcs = destroyStacks[i].PostRun
			}
			if b {
				funcs = append(funcs, s)
				if action == "pre" {
					destroyStacks[i].PreRun = funcs
				} else {
					destroyStacks[i].PostRun = funcs
				}
			}
		}

		if purgeAll {
			purgeObj = true
			purgeEsc = true
			purgeStk = true
		}

		if destroyTarget != "apl" {
			if !purgeObj {
				prompt := "WARNING: purge data in app platform obj buckets? (type YES to confirm)"
				purgeObj = InputPrompt("warn", "YES", prompt)
				if !purgeObj {
					logger.Warn("line:ignoring obj buckets")
				}
			}
			addPrePostRun("pre", "deleteObj", purgeObj, 2)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		var idx int
		for k, i := range destroyStacks {
			i.Path = filepath.Join(paths.Projects, platform.Name, "cmd", i.Name)
			i.GetFullName(ctx)

			if destroyTarget == i.Name {
				idx = k
			}
		}

		switch {
		case idx > 0:
			if st, ok := destroyStacks[idx]; ok {
				stackAction(ctx, st)
			}
		default:
			for i := 1; i < 3; i++ {
				if st, ok := destroyStacks[i]; ok {
					stackAction(ctx, st)
				}
			}
		}

		if purgeEsc {
			org := viper.GetString("pulumiOrg")
			esc := NewEnvObject(org, platform.Name, platform.Stack)

			esc.Remove()
		}
	},
}

func init() {
	// required flags
	destroyCmd.Flags().StringVarP(&platform.Name, "name", "n", "", "APL instance name (required)")
	destroyCmd.MarkFlagRequired("name") //nolint:errcheck
	// optional flags
	destroyCmd.Flags().StringVarP(&destroyTarget, "target", "t", "", "Target a specific project")
	destroyCmd.Flags().BoolVarP(&purgeAll, "purge", "", false, "Purge all infrastructure and Pulumi resources")
	destroyCmd.Flags().BoolVarP(&purgeEsc, "purge-esc", "", false, "Purge Pulumi ESC environment")
	destroyCmd.Flags().BoolVarP(&purgeObj, "purge-obj", "", false, "Purge objects in APL buckets")
	destroyCmd.Flags().BoolVarP(&purgeStk, "purge-stack", "", false, "Purge Pulumi stack data")

	viper.BindPFlags(destroyCmd.LocalFlags()) //nolint:errcheck
}

func stackAction(ctx context.Context, st *MicroStack) {
	switch {
	case purgeAll, purgeStk:
		st.Down(ctx).Remove(ctx)
	default:
		st.Down(ctx)
	}
}

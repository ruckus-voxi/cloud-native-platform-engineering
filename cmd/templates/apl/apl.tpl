package main

import (
	"{{ .repo }}/cmd/apl/app"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "linode")
		aplcfg := config.New(ctx, "apl")
		apl := app.AplResourceInfo{
			Token:  cfg.Require("token"),
			Config: aplcfg,
		}

		err := apl.Run(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

package main

import (
	"{{ .repo }}/cmd/infra/app"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "linode")
		infra := &app.PulumiResourceInfo{
			Token: cfg.Require("token"),
		}

		err := infra.Build(ctx)
		if err != nil {
			return err
		}

		err = infra.Config(ctx)
		if err != nil {
			return err
		}

		return nil
	})
}

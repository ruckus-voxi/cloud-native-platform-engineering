package app

import (
	"fmt"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"

	utils "{{ .repo }}/utils"
)

const (
	aplVersion = "{{ .aplversion }}"
	domainName = "{{ .domain }}"
	email      = "{{ .email }}"
	label      = "{{ .name }}"
	nbTag      = "{{ .nbtag }}"
	objPrefix  = "{{ .objprefix }}"
	region     = "{{ .region }}"
	slug       = "{{ .org }}/{{ .name }}-infra/{{ .stack }}"
)

type AplConfig struct {
	Age        map[string]any    `json:"age"        yaml:"age"`
	ObjBuckets map[string]string `json:"objBuckets" yaml:"objBuckets"`
	ObjKey     map[string]any    `json:"objKey"     yaml:"objKey"`
	Secrets    map[string]any    `json:"secrets"    yaml:"secrets"`
}

type AplResourceInfo struct {
	Resources map[string]any
	Token     string
	Config    *config.Config
}

func (r *AplResourceInfo) Run(ctx *pulumi.Context) error {
	err := run(ctx, r)

	return err
}

func run(ctx *pulumi.Context, r *AplResourceInfo) error {
	// run func
	var stackRef utils.StackRef

	cfg := AplConfig{}

	// esc secrets
	r.Config.RequireObject("age", &cfg.Age)
	r.Config.RequireObject("objBuckets", &cfg.ObjBuckets)
	r.Config.RequireObject("objKey", &cfg.ObjKey)
	r.Config.RequireObject("secrets", &cfg.Secrets)

	// stackref values
	stackRef.Init(ctx, "infraStackRef", slug)

	refs := stackRef.Details("infraStackOutputs")
	_, data := isValid(ctx, refs.SecretValue)
	ipv4, _ := isValid(ctx, data["ipv4"])
	kubecfg, _ := isValid(ctx, data["kubeconfig"])
	nbId, _ := isValid(ctx, data["loadbalancerId"])
	_, subs := isValid(ctx, data["subdomains"])

	// helm: map override values
	objRegion := fmt.Sprintf("%v-1", region)
	override := map[string]any{
		"region":             objRegion,
		"domain":             domainName,
		"token":              r.Token,
		"accessKey":          cfg.ObjKey["accessKey"],
		"secretKey":          cfg.ObjKey["secretKey"],
		"prefix":             objPrefix,
		"buckets":            cfg.ObjBuckets,
		"nodebalancerId":     nbId,
		"nodebalancerIpv4":   ipv4,
		"nodebalancerTag":    nbTag,
		"ageKey":             cfg.Age["publicKey"],
		"agePrivKey":         cfg.Age["privateKey"],
		"lokiAdmin":          cfg.Secrets["lokiAdminPass"],
		"otomiAdmin":         cfg.Secrets["otomiAdminPass"],
		"teamDevelop":        cfg.Secrets["developTeamPass"],
		"platformAdminEmail": email,
		"platformLabel":      label,
	}

	// lke: decode kubeconfig and create new provider
	k, err := utils.DecodeKubeConfig(label, kubecfg, true)
	if err != nil {
		return err
	}

	provider, err := kubernetes.NewProvider(ctx, "aplKubeProvider", &kubernetes.ProviderArgs{
		Kubeconfig: pulumi.String(k),
	})
	if err != nil {
		return err
	}

	ctx.Export("aplKubeProvider", provider)

	// dns: ensure mission critical subdomains are resolving
	at, _ := isValid(ctx, subs["auth"])

	auth, err := utils.NewWaitForDns(ctx, at, &utils.WaitForDnsArgs{
		Domain:  at,
		Ip:      ipv4,
		Timeout: 10,
	})
	if err != nil {
		return err
	}

	kc, _ := isValid(ctx, subs["keycloak"])

	kcloak, err := utils.NewWaitForDns(ctx, kc, &utils.WaitForDnsArgs{
		Domain:  kc,
		Ip:      ipv4,
		Timeout: 10,
	})
	if err != nil {
		return err
	}

	ap, _ := isValid(ctx, subs["api"])

	api, err := utils.NewWaitForDns(ctx, ap, &utils.WaitForDnsArgs{
		Domain:  ap,
		Ip:      ipv4,
		Timeout: 10,
	})
	if err != nil {
		return err
	}

	// helm: deploy apl chart
	values := utils.YamlTemplate(ctx, "./helm/values.tpl", override)
	aplChart := HelmOptions{
		Chart:           "apl",
		DisableWebhooks: false,
		Name:            "apl",
		Lint:            true,
		Repo:            "https://linode.github.io/apl-core",
		ReuseValues:     true,
		Timeout:         1200,
		ValuesFile:      values,
		Version:         aplVersion,
		WaitForJobs:     false,
	}

	_, err = NewKubePkg(ctx, "aplHelmInstall", &KubePkgArgs{
		HelmChart: aplChart,
		Pkg:       "apl-" + aplVersion,
		Provider:  provider,
	}, pulumi.DependsOn([]pulumi.Resource{auth, kcloak, api}),
		pulumi.DeletedWith(provider))
	if err != nil {
		return err
	}

	return nil
}

func isValid(ctx *pulumi.Context, i any) (string, map[string]any) {
	switch v := i.(type) {
	case string:
		if v == "" {
			_ = ctx.Log.Error("string type variable has empty value", nil)
		}

		return v, nil
	case map[string]any:
		if len(v) == 0 {
			_ = ctx.Log.Error("map[string]any type variable has empty value", nil)
		}

		return "", v
	default:
		_ = ctx.Log.Error("type assertion failed", nil)
	}

	return "", nil
}

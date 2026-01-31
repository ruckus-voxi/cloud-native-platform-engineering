package app

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-linode/sdk/v4/go/linode"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"

	utils "{{ .repo }}/utils"
)

const (
	domainName = "{{ .domain }}"
	email      = "{{ .email }}"
	k8sVersion = "{{ .kubeversion }}"
	label      = "{{ .name }}"
	objPrefix  = "{{ .objprefix }}"
	org        = "{{ .org }}"
	nbLabel    = "StaticLoadbalancer"
	nbTag      = "{{ .nbtag }}"
	region     = "{{ .region }}"
	stack      = "{{ .stack }}"
)

var stackOutputMap = pulumi.Map{}

type PulumiResourceInfo struct {
	Data      map[string]string
	Resources ResourceObjects
	Token string
}

type ResourceObjects struct {
	Domain       *linode.Domain
	KubeProvider *kubernetes.Provider
	LkeCluster   *linode.LkeCluster
	LoadBalancer *StaticLoadbalancer
}

type PulumiOpts struct {
	DependsOn   []pulumi.Resource
	DeletedWith pulumi.Resource
}

func (r *PulumiResourceInfo) Build(ctx *pulumi.Context) error {
	err := build(ctx, r)

	return err
}

func (r *PulumiResourceInfo) Config(ctx *pulumi.Context) error {
	err := conf(ctx, r)

	return err
}

func build(ctx *pulumi.Context, r *PulumiResourceInfo) error {
	// cloud infra build func
	objLabels := []string{
		"loki",
		"cnpg",
		"velero",
		"harbor",
		"thanos",
		"tempo",
		"gitea",
	}
	nodeLabels := map[string]string{
		"platform":    label,
		"environment": stack,
	}
	tags := []string{
		{{- range .tags }}
		"{{ . }}",
		{{- end }}
	}

	// obj: create a separate, region scoped key
	objkey, err := linode.NewObjectStorageKey(ctx, "pulumi-obj-key", &linode.ObjectStorageKeyArgs{
		Label: pulumi.String("pulumi-obj-key"),
		Regions: pulumi.StringArray{
			pulumi.String(region),
		},
	})
	if err != nil {
		return err
	}

	env := utils.NewEnvObject(org, label, stack)
	env.Config = map[string]any{
		"apl:objKey": `${objKey}`,
	}

	px := utils.NewPulumixEnvObject("objKey", "secretKey")
	px.MapString = pulumix.Map[string]{
		"accessKey": objkey.AccessKey,
		"secretKey": objkey.SecretKey,
	}
	px.Write(env)

	// obj: provision buckets
	buckets := make(map[string]string)

	for _, bucket := range objLabels {
		bucketName := fmt.Sprintf("%s-%s", objPrefix, bucket)

		_, err = linode.NewObjectStorageBucket(ctx, bucketName, &linode.ObjectStorageBucketArgs{
			AccessKey:      objkey.AccessKey,
			SecretKey:      objkey.SecretKey,
			Region:         pulumi.String(region),
			Label:          pulumi.String(bucketName),
			LifecycleRules: defaultLifecyclePolicy(),
		}, pulumi.DependsOn([]pulumi.Resource{objkey}))
		if err != nil {
			return err
		}

		time.Sleep(1 * time.Second)

		buckets[bucket] = bucketName
	}

	env = utils.NewEnvObject(org, label, stack)
	env.AddConfig(map[string]any{"apl:objBuckets": `${objBuckets}`})
	env.Items = map[int]utils.EscEnvItem{
		1: {Name: "objBuckets", Value: buckets},
	}
	env.Update()

	// dns: create zone
	tagArray := utils.BuildPulumiStringArray(tags)

	domain, err := linode.NewDomain(ctx, domainName, &linode.DomainArgs{
		Type:     pulumi.String("master"),
		Domain:   pulumi.String(domainName),
		SoaEmail: pulumi.String(email),
		Tags:     tagArray,
		TtlSec:   pulumi.Int(30),
	})
	if err != nil {
		if !strings.Contains(err.Error(), "DNS zone already exists") {
			return err
		}

		_ = ctx.Log.Info("domain exists: "+domainName, nil)
	}

	// dns: caa for subdomain wildcard certificate
	_ = domain.ID().ApplyT(func(i string) int {
		id, _ := strconv.Atoi(i)

		_, err = linode.NewDomainRecord(ctx, "wildcardCAA", &linode.DomainRecordArgs{
			DomainId:   pulumi.Int(id),
			RecordType: pulumi.String("CAA"),
			Target:     pulumi.String("letsencrypt.org"),
			Name:       pulumi.String(""),
			Tag:        pulumi.String("issuewild"),
			TtlSec:     pulumi.Int(30),
		}, pulumi.DeletedWith(domain))

		return id
	})

	r.Resources.Domain = domain

	// lke: configure node pools and control plane options
	nodePool := NodePool{
		Autoscaler: true,
		Labels:     nodeLabels,
		Tags:       tags,
	}

	nodePool.SetDefaults()

	aplNodePool := lkeNodePool(nodePool)
	cp := ControlPlane{
		HA: true,
	}
	aplControlPlane := lkeControlPlane(cp)

	// lke: deploy kubernetes cluster
	aplcluster, err := linode.NewLkeCluster(ctx, label, &linode.LkeClusterArgs{
		K8sVersion:   pulumi.String(k8sVersion),
		Label:        pulumi.String(label),
		Pools:        linode.LkeClusterPoolArray{aplNodePool},
		Region:       pulumi.String(region),
		ControlPlane: aplControlPlane,
		Tags:         utils.BuildPulumiStringArray(tags),
	})
	if err != nil {
		return err
	}

	r.Resources.LkeCluster = aplcluster
	stackOutputMap["kubeconfig"] = aplcluster.Kubeconfig

	//lke: create k8s provider
	lkeProvider, err := NewLkeProvider(ctx, "lkeProvider", &LkeProviderArgs{
		Cluster: aplcluster,
		Label:   label,
	}, pulumi.DependsOn([]pulumi.Resource{aplcluster}))
	if err != nil {
		return err
	}

	r.Resources.KubeProvider = lkeProvider.Provider

	return nil
}

func conf(ctx *pulumi.Context, r *PulumiResourceInfo) error {
	// cloud infra config func
	lke := r.Resources.LkeCluster
	lkepv := r.Resources.KubeProvider

	// lke: use provider lookup func to get kube cluster id
	_, ok := lke.ID().ApplyT(func(i string) (int, error) {
		id, _ := strconv.Atoi(i)

		res, err := linode.LookupLkeCluster(ctx, &linode.LookupLkeClusterArgs{
			Id: id,
		})
		if err != nil {
			return 0, err
		}

		retry := 0

		for range 5 {
			if res.Status == "ready" {
				break
			}

			time.Sleep(5 * time.Second)

			retry++
		}

		if res.Status != "ready" && retry == 5 {
			return 0, errors.New("error: timeout waiting for lke cluster")
		}

		envVal := map[string]int{"id": id}
		env := utils.NewEnvObject(org, label, stack)
		env.AddConfig(map[string]any{"lkeId": `${lke.id}`})
		env.Items = map[int]utils.EscEnvItem{
			1: {Name: "lke", Value: envVal},
		}
		env.Update()

		return id, nil
	}).(pulumi.IntOutput)

	if ok {
		// lke: provision static loadbalancer (linode nodebalancer)
		annotations := map[string]string{
			"service.beta.kubernetes.io/linode-loadbalancer-tags":     nbTag,
			"service.beta.kubernetes.io/linode-loadbalancer-preserve": "true",
		}

		// lke: deploy a static loadbalancer to the cluster
		loadbalancer, err := NewStaticLoadbalancer(ctx, nbLabel, &StaticLoadbalancerArgs{
			Annotations: annotations,
			Label:       nbLabel,
			Kubecfg:     label,
		}, pulumi.DependsOn([]pulumi.Resource{lke, lkepv}))
		if err != nil {
			return err
		}

		r.Resources.LoadBalancer = loadbalancer
	}

	// dns: set default dns records for loadbalancer
	domain := r.Resources.Domain
	lb := r.Resources.LoadBalancer
	dnsOpts := PulumiOpts{DependsOn: []pulumi.Resource{lb}}
	dnsRec := func(ip, name, typ string) DnsRecord {
		return DnsRecord{Domain: domain, Opts: dnsOpts, Name: name, RecType: typ, Target: ip}
	}
	subdomains := map[string]string{
		"auth":     "auth." + domainName,
		"keycloak": "keycloak." + domainName,
		"api":      "api." + domainName,
	}

	// dns: root domain ipv4 record
	if lb.Ipv4 != "" {
		err := AddDnsRecord(ctx, dnsRec(lb.Ipv4, "", "A"))
		if err != nil {
			return err
		}

		// dns: subdomains
		for k := range subdomains {
			err = AddDnsRecord(ctx, dnsRec(lb.Ipv4, k, "A"))
		}

		if err != nil {
			return err
		}
	}

	// dns: root domain ipv6 record
	if lb.Ipv6 != "" {
		err := AddDnsRecord(ctx, dnsRec(lb.Ipv6, "", "AAAA"))
		if err != nil {
			return err
		}
	}

	stackOutputMap["subdomains"] = utils.BuildPulumiStringMap(subdomains)

	ctx.Export("infraStackOutputs", stackOutputMap)

	return nil
}

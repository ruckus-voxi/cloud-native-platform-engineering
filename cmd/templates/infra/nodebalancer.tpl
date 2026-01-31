package app

import (
	"time"

	utils "{{ .repo }}/utils"

	"github.com/pulumi/pulumi-linode/sdk/v4/go/linode"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type NodeBalancer struct {
	Id   int
	Ipv4 string
	Ipv6 string
}

type GetNodeBalancerInfo struct {
	InvkOpt pulumi.ResourceOrInvokeOption
	Region  string
	Tag     string
}

func GetNodeBalancer(ctx *pulumi.Context, region, tag string, opt ...any) NodeBalancer {
	var (
		nb      NodeBalancer
		invkOpt pulumi.ResourceOrInvokeOption
	)

	if len(opt) > 1 {
		i := opt[0]
		switch v := i.(type) {
		case *linode.Provider:
			invkOpt = pulumi.Provider(v)
		default:
			invkOpt = nil
		}
	}

	nbinfo := &GetNodeBalancerInfo{
		InvkOpt: invkOpt,
		Region:  region,
		Tag:     tag,
	}

	retry := 0

	for range 5 {
		result := searchNodeBalancer(ctx, nbinfo)
		if len(result.Nodebalancers) > 0 {
			nb.Id = result.Nodebalancers[0].Id
			nb.Ipv4 = result.Nodebalancers[0].Ipv4
			nb.Ipv6 = result.Nodebalancers[0].Ipv6

			break
		}

		time.Sleep(5 * time.Second)

		retry++
	}

	if retry >= 5 && !utils.AssertResource(nb) {
		return NodeBalancer{}
	}

	return nb
}

func searchNodeBalancer(ctx *pulumi.Context, i *GetNodeBalancerInfo) *linode.GetNodebalancersResult {
	matchMethod := "exact"

	res, err := linode.GetNodebalancers(ctx, &linode.GetNodebalancersArgs{
		Filters: []linode.GetNodebalancersFilter{
			{
				Name: "region",
				Values: []string{
					i.Region,
				},
				MatchBy: &matchMethod,
			},
			{
				Name: "tags",
				Values: []string{
					i.Tag,
					"kubernetes",
				},
				MatchBy: &matchMethod,
			},
		},
	}, i.InvkOpt)
	if err != nil {
		return nil
	}

	return res
}

package app

import (
	"errors"
	"slices"
	"strconv"

	"github.com/pulumi/pulumi-linode/sdk/v4/go/linode"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type DnsRecord struct {
	Domain       *linode.Domain
	Name         string
	Opts         PulumiOpts
	RecType      string
	ResourceName string
	Tag          string
	Target       string
	Ttl          int
}

func (d *DnsRecord) SetDefaults() {
	if d.ResourceName == "" {
		if d.Name == "" {
			switch d.RecType {
			case "A":
				d.ResourceName = "defaultIpv4"
			case "AAAA":
				d.ResourceName = "defaultIpv6"
			}
		} else {
			d.ResourceName = d.Name
		}
	}

	if d.Ttl < 30 {
		d.Ttl = 30
	}
}

func GetDomainId(d *linode.Domain) (pulumi.IntOutput, bool) {
	domainId, ok := d.ID().ApplyT(func(i string) (int, error) {
		id, err := strconv.Atoi(i)
		if err != nil {
			return 0, err
		}

		return id, nil
	}).(pulumi.IntOutput)

	return domainId, ok
}

func AddDnsRecord(ctx *pulumi.Context, dns DnsRecord) error {
	dns.SetDefaults()

	id, ok := GetDomainId(dns.Domain)
	if ok {
		_, err := linode.NewDomainRecord(ctx, dns.ResourceName, &linode.DomainRecordArgs{
			DomainId:   id,
			Name:       pulumi.String(dns.Name),
			RecordType: pulumi.String(dns.RecType),
			Tag:        pulumi.String(dns.Tag),
			Target:     pulumi.String(dns.Target),
			TtlSec:     pulumi.Int(dns.Ttl),
		}, pulumi.DependsOn(dns.Opts.DependsOn), pulumi.DeletedWith(dns.Domain))
		if err != nil {
			return err
		}

		return nil
	}

	//nolint:err113
	return errors.New("error: unable to get domain ID")
}

// SearchDomain is a wrapper around linode.LookupDomain, that imports it into
// the Pulumi state if found.
func SearchDomain(ctx *pulumi.Context, domainName string, email string, tags pulumi.StringArray) (*linode.Domain, bool) {
	var domain *linode.Domain

	imported := false

	result, err := linode.LookupDomain(ctx, &linode.LookupDomainArgs{
		Domain: pulumi.StringRef(domainName),
	}, nil)
	if err != nil {
		_ = ctx.Log.Error("error searching for linode domain: "+err.Error(), nil)
	}

	if result.Status == "active" && !slices.Contains(result.Tags, "pulumiImported") {
		domainId := strconv.Itoa(*result.Id)

		domain, err = linode.NewDomain(ctx, domainName, &linode.DomainArgs{
			Type:     pulumi.String("master"),
			Domain:   pulumi.String(domainName),
			SoaEmail: pulumi.String(email),
			Tags:     tags,
			TtlSec:   pulumi.Int(30),
		}, pulumi.Import(pulumi.ID(domainId)))
		if err != nil {
			_ = ctx.Log.Error("error importing linode domain: "+err.Error(), nil)
		} else {
			imported = true
		}
	}

	return domain, imported
}

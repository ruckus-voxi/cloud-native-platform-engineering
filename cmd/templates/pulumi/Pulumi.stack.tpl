environment:
  - {{.name }}/{{ .stack }}
config:
  pulumi:tags:
    pulumi:template: linode-go

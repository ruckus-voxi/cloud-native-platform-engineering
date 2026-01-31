name: {{ .cfgTplName }}
description: App Platform {{ .region }}
runtime: go
config:
  pulumi:tags:
    value:
      pulumi:template: linode-go

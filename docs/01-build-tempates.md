# Build Your Template Library

In deployment steps [#5](../README#5-initialize-your-config) and
[#8](../README#8-add-another-platform) we showed examples of the config file.
You probably observed a default `values` parameter being reused for
both platform definitions. Here is a look at that config file again,
with other params omitted for visibility.

```yaml
defaults:
  - &email ruckus@akamai.com
  - &region nl-ams
  - &values values.tpl  # default defined here
...
platform:
  - name: apl-ams
    domain: ams.arch-linux.io
    email: *email 
    region: *region
    repo: github.com/akamai-developers/aplcli
    values: *values  # used here
  ...
  - name: apl-sea
    domain: sea.arch-linux.io
    email: *email 
    region: *region
    repo: github.com/akamai-developers/aplcli
    values: *values  # used again here
    ...
```

This parameter is telling our CLI which template file to use for
generating a `values.yaml`
[override](https://helm.sh/docs/chart_template_guide/values_files/)
file for the APL Helm chart. In steps [#6](#6-generate-project-code) and [#8](#8-add-another-platform) we showed a
tree view of the local `.aplcli` config directory. Here is that tree
view again with the project code omitted.

```bash
.aplcli
├── config.yaml
└── platforms
    ├── apl-ams
    │   ├── cmd
    │   ├── go.mod
    │   ├── go.sum
    │   └── utils
    ├── apl-sea
    │   ├── cmd
    │   ├── go.mod
    │   ├── go.sum
    │   └── utils
    └── values
        ├── values-example.tpl
        └── values.tpl
```

This default values template was generated when running the `init`
command that created this config directory. The purpose of
templating this versus a hard coding YAML, is so that its reusable
across different platform projects, connected to different Pulumi
ESC environments. As you expand the functionality of this CLI by
adding new commands, refactoring Pulumi code and so on... it's
useful have a single source of truth―one shared directory for all of
your values templates.

The one created by default provides the minimal overrides necessary
our implementation of APL, so it's best to leave it alone, and work
on copies of it for writing new templates. The method also
simplifies versioning templates as you cycle through deployments of
APL (since this CLI tool makes that so painless). For example, we
could have deployed two versions of the Seattle platform―prod and
dev―with similar but slightly different templates for each. Our
config directory could look something like this:

```bash
.aplcli
├── config.yaml
└── platforms
    ├── apl-ams
    │   ├── cmd
    │   ├── go.mod
    │   ├── go.sum
    │   └── utils
    ├── apl-sea
    │   ├── cmd
    │   ├── go.mod
    │   ├── go.sum
    │   └── utils
    └── values
        ├── values-example.tpl
        ├── values.tpl
        ├── seattle-prod-values.tpl
        └── seattle-dev-values.tpl
```

Then platform definitions our config file could something like:

```yaml
...
platform:
  - name: apl-sea-prod
    domain: sea-prod.arch-linux.io
    email: *email 
    region: us-sea
    values: seattle-prod-values.tpl
  - name: apl-sea-dev
    domain: sea-dev.arch-linux.io
    email: *email 
    region: us-sea
    values: seattle-dev-values.tpl
```

The Helm values for APL gives us a *huge* amount of tunables we can
declare to shape the resulting platforms. For example, both our
Seattle and Amsterdam platforms were provisioned with two
teams/tenants (develop and
[admin](https://techdocs.akamai.com/app-platform/docs/platform-teams#admin-team)
along with one custom [platform
admin](https://techdocs.akamai.com/app-platform/docs/platform-user-management#user-roles)
user. If instead we wanted eight different developer teams and six
platform admins in Amsterdam, and four developer teams with 2
platform admins in Seattle, and both with all sorts of messy,
overlapping membership and roles―we would define of that within
values templates.

The `values-example.tpl` file demonstrates a more advanced template
that takes advantaged of many other APL features.

> ![NOTE] \
> Be aware that APL Helm chart [values
schema](https://github.com/linode/apl-core/blob/main/values-schema.yaml)
is known to
[change](https://github.com/linode/apl-core/blob/main/values-changes.yaml)
between releases. Ensure your templates align with your target version
of APL. Pulumi code will let you know if you forget. :)

```yaml
cluster:
  name: {{ .platformLabel }}
  provider: linode
  domainSuffix: {{ .domain }}
otomi:
  adminPassword: {{ .otomiAdmin }}
  hasExternalDNS: true
dns:
  domainFilters: 
    - {{ .domain }}
  provider:
    linode:
      apiToken: {{ .token }}
apps:
  alertmanager:
    enabled: true
  cert-manager:
    issuer: letsencrypt
    stage: production
    email: admin@{{ .domain }}
  grafana:
    enabled: true
  ingress-nginx-platform:
    _rawValues:
      controller:
        service:
          annotations:
            external-dns.alpha.kubernetes.io/ttl: '30'
            service.beta.kubernetes.io/linode-loadbalancer-tags: '{{ .nodebalancerTag }}'
            service.beta.kubernetes.io/linode-loadbalancer-nodebalancer-id: '{{ .nodebalancerId }}'
  jaeger:
    enabled: true
  harbor:
    enabled: true
  knative:
    enabled: true
  loki:
    enabled: true
    adminPassword: {{ .lokiAdmin }}
  otel:
    enabled: true
  prometheus:
    enabled: true
  rabbitmq:
    enabled: true
  tempo:
    enabled: true 
  trivy:
    enabled: true
  kyverno:
    enabled: true
kms:
  sops:
    provider: age
    age:
      publicKey: '{{ .ageKey }}'
      privateKey: '{{ .agePrivKey }}'
obj:
  provider:
    type: linode
    linode:
      region: {{ .region }}
      accessKeyId: {{ .accessKey }}
      secretAccessKey: {{ .secretKey }}
      buckets:
        {{- range .buckets }}
        {{ . }}: {{ $.prefix }}-{{ . }}
        {{- end }}
platformBackups:
  database:
    gitea:
      enabled: true
      pathSuffix: gitea
      retentionPolicy: 7d
      schedule: 0 0 * * *
    harbor:
      enabled: true
      pathSuffix: harbor
      retentionPolicy: 7d
      schedule: 0 1 * * *
    keycloak:
      enabled: true
      pathSuffix: keycloak
      retentionPolicy: 7d
      schedule: 0 2 * * *
  gitea:
    enabled: true
    retentionPolicy: 7d
    schedule: 0 3 * * *
teamConfig:
  develop:
    settings:
      password: {{ .teamDevelop }}
      id: develop
      selfService:
        teamMembers:
          createServices: true
          editSecurityPolicies: false
          useCloudShell: true
          downloadKubeconfig: true
          downloadDockerLogin: false
      managedMonitoring:
        grafana: true
        alertmanager: true
      networkPolicy:
        egressPublic: true
        ingressPrivate: true
      resourceQuota:
        - name: pods
          value: '50'
        - name: services.loadbalancers
          value: '10'
    builds:
      - name: nodejs-hello-world-v0-0-1
        imageName: nodejs-hello-world
        tag: v0.0.1
        trigger: false
        mode:
          docker:
            repoUrl: https://github.com/linode/apl-nodejs-helloworld
            revision: HEAD
            path: ./Dockerfile
          type: docker
      - name: demo-java-maven-v0-0-1
        imageName: demo-java-maven
        tag: v0.0.1
        trigger: false
        mode:
          buildpacks:
            repoUrl: https://github.com/buildpacks/samples
            revision: HEAD
            path: apps/java-maven
          type: buildpacks
    services:
      - headers:
          response:
            set: []
        id: 78595314-cdaf-4b60-acc2-3b1a7f80fe2b
        ingressClassName: platform
        name: httpbin
        ownHost: true
        port: 80
      - id: a106eb22-8c06-41b6-ab15-97aafb0888b5
        ingressClassName: platform
        name: nginx-deployment
        ownHost: true
        paths: []
        port: 80
    workloads:
      - name: nodejs-helloworld
        url: https://github.com/linode/apl-nodejs-helloworld.git
        path: chart/hello-world
        revision: HEAD
      - name: nginx-deployment
        path: k8s-deployment
        revision: main
        selectedChart: k8s-deployment
        url: https://github.com/linode/apl-charts.git
  admin:
    services: []
    workloads:
      - name: nodejs-helloworld
        url: https://github.com/linode/apl-nodejs-helloworld.git
        path: chart/hello-world
        revision: HEAD
files:
  env/teams/develop/workloadValues/nodejs-helloworld.yaml: |
    values: |
      image:
        repository: otomi/nodejs-helloworld
        tag: v1.2.13
  env/teams/develop/workloadValues/nginx-deployment.yaml: |
    values: |
      fullnameOverride: nginx-deployment
      image:
        repository: nginxinc/nginx-unprivileged
        tag: stable
      containerPorts:
        - containerPort: 8080
          protocol: TCP
          name: http
      resources:
        requests:
          cpu: 200m
          memory: 32Mi
      autoscaling:
        minReplicas: 2
        maxReplicas: 10
  env/teams/admin/workloadValues/nodejs-helloworld.yaml: |
    values: |
      image:
        repository: otomi/nodejs-helloworld
        tag: v1.2.13
users:
  - email: {{ .platformAdminEmail }}
    firstName: Ruckus
    lastName: Voxi
    isPlatformAdmin: true
    isTeamAdmin: false
    teams: []
    initialPassword: {{ randInitPass }}
  - email: rthompson@linode.com
    firstName: Ryan
    lastName: Thompson
    isPlatformAdmin: false
    isTeamAdmin: true
    teams:
      - develop
    initialPassword: {{ randInitPass }}
  - email: jennariley@protonmail.com
    firstName: Jenna
    lastName: Riley
    isPlatformAdmin: false
    isTeamAdmin: false
    teams:
      - develop
    initialPassword: {{ randInitPass }}
  - email: amaeve@cryptq.net
    firstName: Anya
    lastName: Maeve
    isPlatformAdmin: false
    isTeamAdmin: false
    teams:
      - develop
    initialPassword: {{ randInitPass }}
```


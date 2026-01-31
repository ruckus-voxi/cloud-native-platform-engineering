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
  harbor:
    enabled: true
  loki:
    enabled: true
    adminPassword: {{ .lokiAdmin }}
  prometheus:
    enabled: true
  tempo:
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
        {{- range $key, $value := .buckets }}
        {{ $key }}: {{ $value }}
        {{- end }}
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

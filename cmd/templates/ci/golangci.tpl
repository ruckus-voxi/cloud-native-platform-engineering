version: "2"

# TODO: add this formatters conf
# formatters:
#   # Enable specific formatter.
#   # Default: [] (uses standard Go formatting)
#   enable:
#     - gci
#     - gofmt
#     - gofumpt
#     - goimports
#     - golines
# issues:
#   exclude-files:

run:
  timeout: 5m
  concurrency: 4
  issues-exit-code: 1

linters:
  default: none
  enable:
    - arangolint
    - asasalint
    - asciicheck
    - bidichk
    - bodyclose
    - canonicalheader
    - containedctx
    - contextcheck
    - copyloopvar
    - decorder
    - dogsled
    - dupl
    - dupword
    - durationcheck
    - embeddedstructfieldcheck
    - errcheck
    - errchkjson
    - errname
    - errorlint
    - exhaustive
    - exptostd
    - fatcontext
    - forbidigo
    - forcetypeassert
    - funcorder
    - ginkgolinter
    - gocheckcompilerdirectives
    - gochecksumtype
    - gocognit
    - goconst
    - gocritic
    - gocyclo
    - godoclint
    - godot
    - godox
    - goheader
    - gomoddirectives
    - gomodguard
    - goprintffuncname
    - gosec
    - gosmopolitan
    - govet
    - grouper
    - iface
    - importas
    - inamedparam
    - ineffassign
    - interfacebloat
    - intrange
    - iotamixing
    - ireturn
    - loggercheck
    - maintidx
    - makezero
    - misspell
    - modernize
    - musttag
    - nakedret
    - nestif
    - nilerr
    - nilnesserr
    - nilnil
    - nlreturn
    - noctx
    - nolintlint
    - nonamedreturns
    - nosprintfhostport
    - paralleltest
    - perfsprint
    - predeclared
    - promlinter
    - protogetter
    - reassign
    - recvcheck
    - rowserrcheck
    - sloglint
    - spancheck
    - sqlclosecheck
    - staticcheck
    - tagalign
    - testableexamples
    - testifylint
    - testpackage
    - thelper
    - tparallel
    - unconvert
    - unparam
    - unqueryvet
    - unused
    - usestdlibvars
    - usetesting
    - varnamelen
    - wastedassign
    - whitespace
    - wsl_v5
    # - zerologlint
  disable:
     - cyclop # TODO: enable and enforce
     - err113
     - depguard
     - exhaustruct
     - funlen
     - gochecknoglobals
     - gochecknoinits
     - lll
     - mirror
     - mnd
     - noinlineerr
     - prealloc
     - revive
     - tagliatelle
     - wrapcheck
     - wsl
  #   - zerologlint
  exclusions:
    presets:
      - common-false-positives
      - legacy
      - std-error-handling
  settings:
    errcheck:
      exclude-functions:
        - ctx.Log.Info(msg, nil)
        - ctx.Log.Error(msg, nil)
    goconst:
      ignore-string-values:
        - 'apl.+'
        - 'pre.+'
        - 'post.+'
        - 'utils.+'
    gosec:
      confidence: medium
    varnamelen:
      check-type-param: true
      ignore-type-assert-ok: true
      ignore-map-index-ok: true
      ignore-chan-recv-ok: true
      ignore-names:
        - err
        - i
        - id
        - ip
        - k
        - n
        - v
        - s
      ignore-decls:
        - b bool
        - c *CodeGenTpl
        - cp ControlPlane
        - e error
        - f *os.File
        - i int
        - m map[string]string
        - m map[string]any
        - nb NodeBalancer
        - np NodePool
        - p *Project
        - l *Logmsg
        - lb *StaticLoadbalancer
        - lm Logmsg
        - r *AplResourceInfo
        - r *PulumiResourceInfo
        - s string
        - t *template.Template
        - t testing.T
    wsl_v5:
      disable:
        - defer
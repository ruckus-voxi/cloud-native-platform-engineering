# Generating API Client

The App Platform is API driven. It adheres to the [OpenAPI 3.0 specification](https://swagger.io/specification/v3/) and exposes an endpoint for users to access the Swagger web UI for documentation and testing―you've probably seen [these](https://petstore.swagger.io/?_gl=1*87igsc*_gcl_au*MTQzMDc3NTkwNS4xNzcxMDMxMDQ1) before. This standardization enables the use of tooling which can consume the JSON schema and auto-generate boilerplate code for us. In the Golang arena, One such tool is the remarkable [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen). In addition to generating HTTP models and server-side code, it can also generate clients.

Start by adding `oapi-codegen` to the arsenal. For systems running Go 1.24+, the official project recommends installing with `go get -tool`, but the `go install` method works as well. We won't dig into the reasons for picking one or the other, but if you're interested, a co-maintainer of the tool explains it pretty well in this [post](https://www.jvt.me/posts/2025/01/27/go-tools-124/).  

```bash
go get -tool github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

# alternatively use `go install`
go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
```

> [!IMPORTANT]
> These lab instructions currently support APL `v4.12.1` (the default version deployed by APL CLI). Updates are still pending for `v4.14.1`, which does not expose the API JSON schema at `/api/api-docs/`.

Next, if you haven't already, login to your APL console dashboard at `https://console.${DOMAIN}`. Be sure to replace `${DOMAIN}` with the actual domain/subdomain of your APL installation. Once logged in you can visit the Swagger UI at `https://console.${DOMAIN}/api/api-docs/swagger/`. Look around for bit and get familiar, you'll notice there are both `/v1` and `/v2` endpoints. The differences are not immediately clear, but for a short example, you'll find API definitions for _team-specific_ configuration from underneath of `/v1/teams/{teamid}` (images, workloads, etc) or _platform specific_ settings from `/v2` or `/v1`. With some rare exceptions (when the API is not finalized or there are new apps under testing), the entire values structure is represented in these API definitions.

Remove `/swagger/` from the URL path and reload the page, so that you are visiting `https://console.${DOMAIN}/api/api-docs/`. You should see the massive blob of JSON representing every possible API endpoint. Highlight all of it and copy/paste to a file called `api.yaml`.

Next you'll create an `oapi-codegen` config file called `cfg.yaml`. The values for `package` and `output` can be customized if you'd like, but for simplicity let's just stick with the convention in the example below.

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/oapi-codegen/oapi-codegen/HEAD/configuration-schema.json
package: client
output: client.go
generate:
  models: true
  client: true
```

Now provide these two `yaml` files you just created to the `oapi-codegen` command, and watch your new `client.go` file come to life. The part we are most interested in starts at around line `4442` with the `Client` struct.

```bash
oapi-codegen -config cfg.yaml api.yaml
```

For reference, the `aplcli` directory tree looks like this after a fresh `git clone` and `cd` into it.

```bash
.
├── cmd
│   └── templates
│       ├── apl
│       ├── ci
│       ├── infra
│       ├── init
│       ├── pulumi
│       ├── utils
│       └── values
├── config
├── docs
│   └── solutions
└── media
    ├── images
    └── screencasts
```

Lastly, move or copy the newly generated `client.go` to `internal/client`, where it can be imported as a private library APL CLI commands. Assuming you currently in the repo root, and you ran the previous `oapi-codgen` command from one directory outside of it, the two commands you need to run are:

```bash
mkdir cmd/client
cp ../client.go cmd/client/
```

Now before we start importing this module into our CLI code, we need to quickly install a few more dependencies, the first being another package by the `oapi-codegen` project called `securityprovider`. This provides a `RequestEditorFn` type our client uses to intercept and mutate HTTP requests―we use it for [Bearer Authentication](https://swagger.io/docs/specification/v3_0/authentication/bearer-authentication/) in this case. See the [Authenticated API Example](https://github.com/oapi-codegen/oapi-codegen/blob/main/examples/authenticated-api/README.md) for additional info. 

```bash
go get github.com/oapi-codegen/oapi-codegen/v2/pkg/securityprovider
```

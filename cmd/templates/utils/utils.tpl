package internal

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pulumi/pulumi-command/sdk/go/command/local"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v2"
)

type WaitForDns struct {
	pulumi.ResourceState

	IsReady bool `pulumi:"dnsReady"`
}

type WaitForDnsArgs struct {
	Domain  string
	Ip      string
	Timeout int
}

func RandInitPass() string {
	return uuid.NewString()
}

//nolint:unconvert
func DecodeKubeConfig(label, cfg string, write bool) (string, error) {
	kcfg, err := base64.StdEncoding.DecodeString(string(cfg))
	if err != nil {
		return "", err
	}

	// write kubeconfig to filesystem
	if write {
		var data map[string]any

		err := yaml.Unmarshal(kcfg, &data)
		if err != nil {
			return "", err
		}

		homeDir := os.Getenv("HOME")
		fileName := label + "-kubeconfig.yaml"
		file := filepath.Join(homeDir, ".kube", fileName)

		err = WriteFile(file, kcfg)
		if err != nil {
			return "", err
		}
	}

	return string(kcfg), nil
}

func YamlTemplate(ctx *pulumi.Context, tpl string, values map[string]any, write ...bool) string {
	_, file := filepath.Split(tpl)

	funcMap := template.FuncMap{
		"randInitPass": RandInitPass,
	}

	t, err := template.New(file).Funcs(funcMap).ParseFiles(tpl)
	if err != nil {
		_ = ctx.Log.Error("error parsing values template file: "+err.Error(), nil)
	}

	buf := &bytes.Buffer{}
	if err := t.Execute(buf, values); err != nil {
		_ = ctx.Log.Error("error executing template file: "+err.Error(), nil)
	}

	if write != nil && write[0] {
		var data map[string]any
		if err := yaml.Unmarshal(buf.Bytes(), &data); err != nil {
			_ = ctx.Log.Error(err.Error(), nil)
		}

		yamlFile := strings.Replace(tpl, ".tpl", ".yaml", 1)
		if err := WriteFile(yamlFile, buf.Bytes()); err != nil {
			_ = ctx.Log.Error(err.Error(), nil)
		}
	}

	return buf.String()
}

func WriteFile(file string, data []byte) error {
	dir, _ := filepath.Split(file)

	if _, err := os.Stat(dir); !os.IsExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	} else {
		return err
	}

	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err = io.Writer.Write(f, data); err != nil {
		return err
	}

	return f.Sync()
}

func AssertResource(items ...any) bool {
	var result bool

	chk := func(i any) bool {
		v := reflect.ValueOf(i)

		if v.Kind() == reflect.Map {
			for _, e := range v.MapKeys() {
				val := v.MapIndex(e)
				switch t := val.Interface().(type) {
				case bool:
					if !t {
						return false
					}
				default:
					return true
				}
			}
		}

		if v.IsValid() && !v.IsZero() {
			return true
		}

		return false
	}

	if len(items) > 1 {
		for _, i := range items {
			result = chk(i)
			if !result {
				break
			}
		}

		return result
	}

	result = chk(items[0])

	return result
}

func ResolveDns(domain string, ip string, timeout int) (string, error) {
	var addr string

	start := time.Now()
	end := time.Duration(timeout) * time.Minute
	check := func(err error) error { // old: check := func(err error, t int) error {
		conditions := []bool{
			!strings.Contains(err.Error(), "no such host"),
			!strings.Contains(err.Error(), "server misbehaving"),
		}
		if conditions[0] && conditions[1] {
			return err
		}

		return nil
	}

	resolver := func(ctx context.Context, proto string, address string) (net.Conn, error) {
		d := net.Dialer{}

		return d.DialContext(ctx, "udp", "8.8.8.8:53")
	}

	dns := &net.Resolver{
		PreferGo: true,
		Dial:     resolver,
	}

	for time.Since(start) < end && addr != ip {
		result, err := dns.LookupIPAddr(context.Background(), domain)
		if err != nil {
			// return "", check(err, timeout)  old
			return "", check(err)
		}

		if len(result) > 0 {
			addr = result[0].IP.String()
		}

		time.Sleep(5 * time.Second)
	}

	if time.Since(start) >= end {
		return addr, fmt.Errorf("exceded timeout waiting for %s to resolve", domain)
	}

	return addr, nil
}

func NewWaitForDns(ctx *pulumi.Context, name string, args *WaitForDnsArgs, opts ...pulumi.ResourceOption) (*WaitForDns, error) {
	// component resource wrapper around the ResolveDns func
	// use this to control the aggressive parrallelism, or to track resolution in pulumi state
	var dnsresource WaitForDns

	err := ctx.RegisterComponentResource("pkg:index:WaitForDns", name, &dnsresource, opts...)
	if err != nil {
		return nil, err
	}

	ipAddr, err := ResolveDns(args.Domain, args.Ip, args.Timeout)
	if err != nil {
		return nil, err
	}

	if ipAddr != "" {
		msg := "resolved IP is: " + ipAddr
		_ = ctx.Log.Info(msg, nil)
		dnsresource.IsReady = true
	}

	return &dnsresource, nil
}

// pulumi helpers

func NullResource(ctx *pulumi.Context) *local.Command {
	n, err := local.NewCommand(ctx, "nullCommand", &local.CommandArgs{
		Create: pulumi.String(":"),
		Interpreter: pulumi.StringArray{
			pulumi.String("/bin/bash"),
			pulumi.String("-c"),
		},
	})
	if err != nil {
		msg := fmt.Sprintf("error executing null resource command %v", err)
		_ = ctx.Log.Error(msg, nil)
	}

	return n
}

//nolint:unconvert
func BuildPulumiStringArray(s []string) pulumi.StringArray {
	var strArray pulumi.StringArray

	for _, str := range s {
		t := pulumi.String(str)
		strArray = append(strArray, pulumi.String(t))
	}

	return strArray
}

func BuildPulumiStringMap(sm map[string]string) pulumi.StringMap {
	strMap := pulumi.StringMap{}
	for k, v := range sm {
		strMap[k] = pulumi.String(v)
	}

	return strMap
}

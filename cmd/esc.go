package cmd

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"maps"
	"slices"
	"strings"

	age "filippo.io/age"
	esc "github.com/pulumi/esc-sdk/sdk/go"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumix"
)

type EscEnv struct {
	EnvName    string
	Config     map[string]any
	Items      map[int]EscEnvItem
	OrgName    string
	ProjName   string
	PulumixMap pulumix.Map[string]
	Objects    []EscEnvItem
}

type EscEnvItem struct {
	Name  string
	Value any
}

type PulumixEscEnvItem struct {
	Name       string
	MapString  pulumix.Map[string]
	SecretKeys []string
}

func NewEnvObject(org, proj, stack string) EscEnv {
	env := EscEnv{
		EnvName:  stack,
		OrgName:  org,
		ProjName: proj,
	}

	return env
}

func EscExists(orgName, projName, envName string) bool {
	authCtx, escClient, err := esc.DefaultLogin()
	if err != nil {
		logger.Error("auth to check if esc environment exists: " + err.Error())
	}

	_, _, err = escClient.GetEnvironment(authCtx, orgName, projName, envName)

	return err == nil
}

func (e *EscEnv) Init() {
	authCtx, escClient, err := esc.DefaultLogin()
	if err != nil {
		logger.Error("pulumi esc init login: " + err.Error())
	}

	err = escClient.CreateEnvironment(authCtx, e.OrgName, e.ProjName, e.EnvName)
	if err != nil {
		if strings.Contains(err.Error(), "409 Conflict") {
			logger.Info("esc environment already initialized")

			return
		} else {
			logger.Error("initialize pulumi esc environment: " + err.Error())
		}
	}

	v := make(map[string]any)
	values := e.BuildValues(v)

	if err := e.Write(values); err != nil {
		logger.Error("write initial esc environment: " + err.Error())
	}
}

func (e *EscEnv) AddConfig(c map[string]any) *EscEnv {
	e.Config = c

	return e
}

func (e *EscEnv) GetConfig(key string) (string, int) {
	authCtx, escClient, err := esc.DefaultLogin()
	if err != nil {
		logger.Error("auth to esc environment for update: " + err.Error())
	}

	_, config, err := escClient.OpenAndReadEnvironment(authCtx, e.OrgName, e.ProjName, e.EnvName)
	if err != nil {
		logger.Error("open and read esc environment: " + err.Error())
	}

	value, ok := config[key]
	if !ok {
		msg := fmt.Sprintf("\"%s\" not found", key)
		logger.Error("get value from esc environment: " + msg)
	}

	switch v := value.(type) {
	case string:
		return v, 0
	case int:
		return "", v
	default:
		return "", 0
	}
}

func (e *EscEnv) Update() {
	authCtx, escClient, err := esc.DefaultLogin()
	if err != nil {
		logger.Error("auth to esc environment for update: " + err.Error())
	}

	env, _, err := escClient.GetEnvironment(authCtx, e.OrgName, e.ProjName, e.EnvName)
	if err != nil {
		logger.Error("get existing esc environment: " + err.Error())
	}

	// merge new and existing pulumiConfig maps
	c := env.Values.PulumiConfig
	maps.Copy(c, e.Config)
	e.Config = c

	// build new values map from new and existing
	v, err := env.GetValues().ToMap()
	if err != nil {
		logger.Error("get existing esc pulumi config: " + err.Error())
	}

	values := v

	switch {
	case len(e.Objects) > 0:
		for _, i := range e.Objects {
			values[i.Name] = i.Value
		}
	case len(e.Items) > 0:
		values = e.BuildValues(values)
	}

	if err := e.Write(values); err != nil {
		logger.Error("write update to esc environment: " + err.Error())
	}
}

func (e *EscEnv) Write(values map[string]any) error {
	authCtx, escClient, err := esc.DefaultLogin()
	if err != nil {
		return err
	}

	payload := &esc.EnvironmentDefinition{
		Values: &esc.EnvironmentDefinitionValues{
			PulumiConfig:         e.Config,
			AdditionalProperties: values,
		},
	}

	_, err = escClient.UpdateEnvironment(authCtx, e.OrgName, e.ProjName, e.EnvName, payload)
	if err != nil {
		return err
	}

	return nil
}

func (e *EscEnv) Remove() {
	authCtx, escClient, err := esc.DefaultLogin()
	if err != nil {
		logger.Error("pulumi esc remove login: " + err.Error())
	}

	err = escClient.DeleteEnvironment(authCtx, e.OrgName, e.ProjName, e.EnvName)
	if err != nil {
		logger.Error("delete esc environment: " + err.Error())
	}

	logger.Info("purged esc environment")
}

func (e *EscEnv) BuildValues(val map[string]any) map[string]any {
	isValid := func(i any) bool {
		switch i.(type) {
		case map[string]string, map[string]any, map[string]int:
			return true
		default:
			return false
		}
	}

	for _, v := range e.Items {
		if isValid(v.Value) {
			val[v.Name] = v.Value
		}
	}

	return val
}

func NewPulumixEnvObject(name string, secrets ...string) PulumixEscEnvItem {
	px := PulumixEscEnvItem{
		Name:       name,
		SecretKeys: secrets,
	}

	return px
}

func (px *PulumixEscEnvItem) Write(env EscEnv) {
	px.MapString.AsAny().ApplyT(func(i any) error {
		v, ok := i.(map[string]string)
		if !ok {
			logger.Error("type assertion failed: wants map[string]string")
		}

		m := make(map[string]any)

		for key, val := range v {
			if slices.Contains(px.SecretKeys, key) {
				m[key] = FnSecret(val)
			} else {
				m[key] = val
			}
		}

		item := EscEnvItem{
			Name:  px.Name,
			Value: m,
		}
		env.Items = BuildEscItems(item)
		env.Update()

		return nil
	})
}

func Passgen() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		logger.Error("passgen random bytes: " + err.Error())
	}

	return base64.URLEncoding.EncodeToString(b)
}

func GenAgeKeys() (*age.X25519Identity, error) {
	ageKeys, err := age.GenerateX25519Identity()
	if err != nil {
		return nil, err
	}

	return ageKeys, nil
}

func BuildEscItems(i any) map[int]EscEnvItem {
	items := make(map[int]EscEnvItem)

	switch v := i.(type) {
	case []EscEnvItem:
		for idx, item := range v {
			items[idx] = item
		}
	case EscEnvItem:
		items[1] = v
	}

	return items
}

func FnSecret(i string) map[string]string {
	return map[string]string{
		"fn::secret": i,
	}
}

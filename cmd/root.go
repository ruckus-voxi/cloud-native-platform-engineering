package cmd

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

const (
	confDir   = ".aplcli"
	idpDir    = ".aplcli/platforms"
	valuesDir = ".aplcli/platforms/values"
	version   = "1.0.0"
)

type ProjectPaths struct {
	Config   string
	Projects string
	Values   string
}

//go:embed templates/*
var templates embed.FS

var (
	cfgFile    string
	cfgArray   []map[string]any
	cfgIndex   int
	paths      ProjectPaths
	valuesFile string
)

var rootCmd = &cobra.Command{
	Use:     "aplcli",
	Short:   "Cloud Native Platform Engineering",
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := loadProjConfig(); err != nil {
			logger.Error(err.Error())

			return err
		}

		return nil
	},
}

// top-level subcommands

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new aplcli environment",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		path := projPath()
		err := os.MkdirAll(path.Projects, 0754)
		if err != nil {
			logger.Error("project directory creation: " + err.Error())
		}

		err = os.MkdirAll(path.Values, 0754)
		if err != nil {
			logger.Error("values directory creation: " + err.Error())
		}

		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		domain := SetupPrompt("Domain")
		email := SetupPrompt("Email")
		name := SetupPrompt("Platform name")
		region := SetupPrompt("Region")

		name = strings.ReplaceAll(name, "_", "-")
		defaultRepo := "github.com/akamai-developers/aplcli"
		org := GetPulumiUser()

		buf := &bytes.Buffer{}
		data := map[string]any{
			"domain": domain,
			"email":  email,
			"name":   name,
			"org":    org,
			"region": region,
			"repo":   defaultRepo,
			"values": valuesFile,
		}

		t := template.Must(template.New("cfg.tpl").ParseFS(templates, "templates/init/cfg.tpl"))
		if err := t.Execute(buf, &data); err != nil {
			logger.Info("execute init template: " + err.Error())
		}

		// write config file
		cfgFile := filepath.Join(paths.Config, "config.yaml")
		if err := os.WriteFile(cfgFile, buf.Bytes(), 0600); err != nil {
			logger.Error("write config file: " + err.Error())
		}

		logger.Info("config file written to: " + cfgFile)

		// write example helm values template
		fsDir := "templates/values"
		exTpl := "values-example.tpl"
		fname := filepath.Join(paths.Values, exTpl)
		v, err := templates.ReadFile(fsDir + "/" + exTpl)
		if err != nil {
			msg := fmt.Sprintf("read %s: %s", exTpl, err.Error())
			logger.Error(msg)
		}

		if err := os.WriteFile(fname, v, 0600); err != nil {
			msg := fmt.Sprintf("write %s: %s", exTpl, err.Error())
			logger.Error(msg)
		}

		// write default helm values template
		tplPath := filepath.Join(fsDir, valuesFile)
		tpl := template.Must(template.New(valuesFile).ParseFS(templates, tplPath))

		fname = filepath.Join(paths.Values, valuesFile)
		v = []byte(tpl.Root.String())

		if err := os.WriteFile(fname, v, 0600); err != nil {
			msg := fmt.Sprintf("write %s template file: %s", valuesFile, err.Error())
			logger.Error(msg)
		}

		valuesFullPath := filepath.Join(paths.Values, valuesFile)
		logger.Info("default values.tpl written to: " + valuesFullPath)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig, PreChk)

	paths = projPath()
	valuesFile = "values.tpl"

	// global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.aplcli/config.yaml)")
	rootCmd.PersistentFlags().SetNormalizeFunc(nameNormalizeFunc)

	// subcommands
	rootCmd.AddCommand(createCmd, deployCmd, destroyCmd, initCmd)

	// usage func
	helpText(rootCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config file with name "config" (without extension).
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config/")
		viper.AddConfigPath(home + "/.aplcli/")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		logger.Info("Using config file: " + viper.ConfigFileUsed())
	}
}

func projPath() ProjectPaths {
	var paths ProjectPaths

	home, err := os.UserHomeDir()
	if err != nil {
		msg := "locate user home directory: " + err.Error()
		logger.Error(msg)
	}

	paths.Config = filepath.Join(home, confDir)
	paths.Projects = filepath.Join(home, idpDir)
	paths.Values = filepath.Join(home, valuesDir)

	return paths
}

// loadProjConfig parses the array of platform definitions in config.yaml to
// load the definition matching the value provided by the required --name flag.
// This ensures loadig of the correct definition on each invokation.
func loadProjConfig() error {
	cfg, ok := viper.AllSettings()["platform"].([]any)
	if len(cfg) == 0 {
		return nil
	}

	if !ok {
		return errors.New("type assertion failed: wants []any")
	}

	for _, i := range cfg {
		if c, ok := i.(map[string]any); ok {
			cfgArray = append(cfgArray, c)
		}

		if len(cfgArray) < 1 {
			return errors.New("load platform config: no valid config was found")
		}
	}

	for idx, i := range cfgArray {
		if platform.Name == i["name"] {
			cfgIndex = idx
		}
	}

	runCfg, err := yaml.Marshal(cfgArray[cfgIndex])
	if err != nil {
		return errors.New("yaml marshal running config: " + err.Error())
	}

	if err := yaml.Unmarshal(runCfg, &platform); err != nil {
		return errors.New("yaml unmarshal running config: " + err.Error())
	}

	return nil
}

// nameNormalizeFunc removes hyphens from a flag name and returns a lowercased
// string name. A flag such as --node-type is converted to a normalized name of
// nodetype, which aligns with how viper returns matching keys found in the
// config file. Keys are camel cased in the config file for readability,
// but then lowercased when unmarshaled to the platform struct.
func nameNormalizeFunc(f *pflag.FlagSet, name string) pflag.NormalizedName {
	normalizedName := strings.ReplaceAll(name, "-", "")

	return pflag.NormalizedName(normalizedName)
}

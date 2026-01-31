package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var rcloneHeader = `
// This file is used by the aplicli tool that generated this project, not by the
// project itself, but is included here as a template or starting point for
// future rclone operations a user may want to implement.`

type Project struct {
	Data     Platform
	Base     string
	Dirs     map[string][]string
	Name     string
	Packages []string
	Values   string
	Repo     string
}

type CodeGenTpl struct {
	Data  map[string]any
	Helm  bool
	Name  string
	Pkg   bool
	Paths map[string]string
}

func NewProject(p Platform, projPath string, values string) Project {
	proj := Project{
		Data: p,
		Base: projPath,
		Name: p.Name,
		Packages: []string{
			"apl",
			"infra",
			"utils",
		},
		Values: values,
	}

	return proj
}

func (pr *Project) Init(ctx context.Context) {
	if err := os.Chdir(pr.Base); err != nil {
		logger.Error("change to project root directory: " + err.Error())
	}

	mod := platform.Repo
	cmd := exec.CommandContext(ctx, "go", "mod", "init", mod)

	stdout, err := cmd.CombinedOutput()
	if err != nil {
		if !strings.Contains(string(stdout), "already exists") {
			logger.Error("go project init: " + err.Error())
		}
	}

	if string(stdout) != "" {
		logger.Info("go mod init")
	}

	if _, err := os.Stat("go.mod"); err == nil {
		cmd := exec.CommandContext(ctx, "go", "mod", "tidy")

		stdout, err := cmd.CombinedOutput()
		if err != nil {
			logger.Error(string(stdout))
			logger.Error("go mod tidy: " + err.Error())
		}

		if string(stdout) != "" {
			logger.Info("go mod tidy")
		}
	}
}

func (p *Project) CodeGen() {
	codegen(p)
}

func codegen(p *Project) {
	var (
		srcFile  CodeGenTpl
		genFiles []string
	)

	// write golangci.yaml to project root

	var cidata map[string]any

	if err := dirGen(p.Base); err != nil {
		logger.Error("create project directory: " + err.Error())
	}

	buf := &bytes.Buffer{}

	golangci := template.Must(template.New("golangci.tpl").ParseFS(templates, "templates/ci/golangci.tpl"))
	if err := golangci.Execute(buf, &cidata); err != nil {
		logger.Info("execute golangci.yaml template: " + err.Error())
	}

	ciYAML := filepath.Join(p.Base, ".golangci.yaml")
	if err := os.WriteFile(ciYAML, buf.Bytes(), 0600); err != nil {
		logger.Error("write golangci.yaml: " + err.Error())
	}

	genFiles = append(genFiles, ciYAML)

	for _, i := range p.Packages {
		data := tplParser(p.Data)

		data["org"] = viper.GetString("pulumiOrg")
		data["cfgTplName"] = p.Name

		cmdPath := filepath.Join(p.Base, "cmd", i)
		pkg := cmdPath + "/app"
		helm := cmdPath + "/helm"
		pkgPaths := map[string]string{"cmd": cmdPath, "base": p.Base, "pkg": pkg}

		switch i {
		case "apl":
			data["cfgTplName"] = p.Name

			if err := dirGen(helm, pkg); err != nil {
				logger.Error("creating apl pkg directories: " + err.Error())
			}

			dst := filepath.Join(helm, "values.tpl")
			writeSrcFile(p.Values, dst)

			genFiles = append(genFiles, dst)

			srcFile = CodeGenTpl{
				Data:  data,
				Helm:  true,
				Name:  i,
				Paths: pkgPaths,
				Pkg:   true,
			}
		case "infra":
			data["cfgTplName"] = p.Name + "-infra"

			if err := dirGen(pkg); err != nil {
				logger.Error("creating infra pkg directories: " + err.Error())
			}

			srcFile = CodeGenTpl{
				Data:  data,
				Name:  i,
				Paths: pkgPaths,
				Pkg:   true,
			}
		default:
			internalPkg := filepath.Join(p.Base, i)
			if err := dirGen(internalPkg); err != nil {
				msg := fmt.Sprintf("creating %s directories: %s", i, err.Error())
				logger.Error(msg)
			}

			srcFile = CodeGenTpl{
				Data:  data,
				Name:  i,
				Paths: pkgPaths,
			}
		}

		files := writeTemplates(&srcFile)
		genFiles = append(genFiles, files...)
	}

	logger.Info("code generated at:")

	for _, i := range genFiles {
		logger.Info("line:" + i)
	}
}

func getPulumiTpls(data map[string]any, path string) []string {
	dir, err := templates.ReadDir("templates/pulumi")
	if err != nil {
		logger.Error("pulumi templates: " + err.Error())
	}

	p := filepath.Join("templates", "pulumi", "*.tpl")
	files := make([]string, 0, 2)

	for _, i := range dir {
		buf := &bytes.Buffer{}
		tpl := template.Must(template.New(i.Name()).ParseFS(templates, p))

		if err := tpl.Execute(buf, &data); err != nil {
			logger.Error("get pulumi config templates: " + err.Error())
		}

		fname := strings.TrimSuffix(tpl.Name(), ".tpl")
		if strings.Contains(fname, "stack") {
			fname = strings.NewReplacer("stack", platform.Stack).Replace(fname)
		}

		src := filepath.Join(path, fname+".yaml")

		if err := os.WriteFile(src, buf.Bytes(), 0600); err != nil {
			logger.Error("write pulumi template files: " + err.Error())
		}

		files = append(files, src)
	}

	return files
}

func dirGen(dirs ...string) error {
	// add src directories
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0754); err != nil {
			return err
		}
	}

	return nil
}

func writeTemplates(c *CodeGenTpl) []string {
	var (
		src       string
		files     []string
		confFiles []string
	)

	//nolint:gocritic
	funcMap := template.FuncMap{
		"randInitPass": func() string { return uuid.NewString() },
		"rcloneHeader": func() string { return strings.TrimSpace(rcloneHeader) },
	}

	tmpls, err := templates.ReadDir("templates/" + c.Name)
	if err != nil {
		logger.Error("read templates: " + err.Error())
	}

	tplPath := filepath.Join("templates", c.Name, "*.tpl")

	for _, t := range tmpls {
		buf := &bytes.Buffer{}
		tpl := template.Must(template.New(t.Name()).Funcs(funcMap).ParseFS(templates, tplPath))

		if err := tpl.Execute(buf, &c.Data); err != nil {
			logger.Error("execute parsed project templates: " + err.Error())
		}

		tplName := strings.TrimSuffix(tpl.Name(), ".tpl")
		switch {
		case c.Pkg && tplName == c.Name:
			tplName = strings.NewReplacer(tplName, "main").Replace(tplName)
			src = filepath.Join(c.Paths["cmd"], tplName+".go")

			confFiles = getPulumiTpls(c.Data, c.Paths["cmd"])
			if len(confFiles) < 2 {
				msg := fmt.Sprintf("get pulumi config templates: wants 2, got %d", len(confFiles))
				logger.Error(msg)
			}

			files = append(files, confFiles...)
		case c.Pkg:
			src = filepath.Join(c.Paths["pkg"], tplName+".go")
		default:
			src = filepath.Join(c.Paths["base"], c.Name, tplName+".go")
		}

		if err := os.WriteFile(src, buf.Bytes(), 0600); err != nil {
			logger.Error("write src template files: " + err.Error())
		}

		files = append(files, src)
	}

	return files
}

func writeSrcFile(src, dst string) {
	f, err := os.ReadFile(src)
	if err != nil {
		logger.Error(err.Error())
	}

	if err := os.WriteFile(dst, f, 0600); err != nil {
		msg := "writing source file " + src
		logger.Error(msg)
	}
}

func tplParser(p Platform) map[string]any {
	yamlData, err := yaml.Marshal(p)
	if err != nil {
		logger.Error(err.Error())
	}

	data := make(map[string]any)

	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		logger.Error("unmarshal template data for codegen: " + err.Error())
	}

	return data
}

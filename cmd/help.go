package cmd

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/galactixx/stringwrap"
	"github.com/spf13/cobra"
)

var lgtxt = `
:::'###::::'########::'##:::::::::::'######::'##:::::::'####:
::'## ##::: ##.... ##: ##::::::::::'##... ##: ##:::::::. ##::
:'##:. ##:: ##:::: ##: ##:::::::::: ##:::..:: ##:::::::: ##::
'##:::. ##: ########:: ##:::::::::: ##::::::: ##:::::::: ##::
 #########: ##.....::: ##:::::::::: ##::::::: ##:::::::: ##::
 ##.... ##: ##:::::::: ##:::::::::: ##::: ##: ##:::::::: ##::
 ##:::: ##: ##:::::::: ########::::. ######:: ########:'####:
..:::::..::..:::::::::........::::::......:::........::....::
`
var replacer = strings.NewReplacer(
	":", fmt.Sprintf("%s%s%s", Grey, ":", Reset),
	".", fmt.Sprintf("%s%s%s", DarkGrey, ".", Reset),
	"'", fmt.Sprintf("%s%s%s", Grey, "'", Reset),
	"#", fmt.Sprintf("%s%s%s", Green, "#", Reset),
)

var banner = replacer.Replace(lgtxt)

var versionTpl = `{{printf "aplcli version %s" .Version | helpTxt }}`

var usageTpl = `{{ headingTxt "Usage:" }}{{if .Runnable}}
  {{ cmdTxt (.UseLine) }}{{end}}{{if .HasAvailableSubCommands}}
  {{cmdTxt (.CommandPath) }} {{ cmdTxt "[command]" }}{{end}}{{if gt (len .Aliases) 0}}

{{ headingTxt "Aliases:" }}:
  {{ cmdTxt (.NameAndAliases) }}{{end}}{{if .HasExample}}

{{ headingTxt "Examples:" }}
{{ cmdTxt (.Example) }}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

{{ headingTxt "Available Commands:"}}{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpadTxt .Name .NamePadding }} {{ cmdTxt .Short }}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{ headingTxt .Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpadTxt .Name .NamePadding }} {{cmdTxt .Short }}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

{{ headingTxt "Additional Commands:" }}{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpadTxt .Name .NamePadding }} {{ cmdTxt .Short }}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{ headingTxt "Flags:" }}
{{ cmdTxt .LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

{{ headingTxt "Global Flags:" }}
{{ cmdTxt .InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

{{ headingTxt "Additional help topics:" }}{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpadTxt .CommandPath .CommandPathPadding}} {{ cmdTxt .Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

{{ .CommandPath |  moreInfo | cmdTxt }}{{end}}
`

func helpText(cmd *cobra.Command) {
	var helpTpl = cmd.HelpTemplate()

	tplFuncs := template.FuncMap{
		"headingTxt": func(s string) string {
			return fmt.Sprintf("%s%s%s", Magenta, s, Reset)
		},
		"cmdTxt": func(s string) string {
			return fmt.Sprintf("%s%s%s", Grey, s, Reset)
		},
		"rpadTxt": func(s string, padding int) string {
			tpl := fmt.Sprintf("%s%%-%ds%s", Grey, padding, Reset)

			return fmt.Sprintf(tpl, s)
		},
		"moreInfo": func(s string) string {
			tpl := fmt.Sprintf(`Use "%s [command] --help" for more information about a command.`, s)

			return tpl
		},
		"helpTxt": func(s string) string {
			sep := "============================================================="
			txt := fmt.Sprintf("%s%s%s%s\n", banner, Grey, sep, Reset)

			pad := len(sep) - len(s)
			if pad > 0 {
				pad /= 2
			} else {
				pad = 0
			}

			padding := strings.Repeat(" ", pad)

			// wrap text if string is longer than separator
			if len(s) > len(sep) {
				s, _, _ = stringwrap.StringWrap(s, len(sep), 0, false)
			}

			txt += fmt.Sprintf("%s%s%s%s%s", Grey, padding, s, padding, Reset)

			return txt
		},
	}

	cobra.AddTemplateFuncs(tplFuncs)

	helpTpl = strings.NewReplacer(
		`{{. | trimTrailingWhitespaces}}`, `{{ . | helpTxt | trimTrailingWhitespaces }}`,
	).Replace(helpTpl)

	cmd.SetVersionTemplate(versionTpl)
	cmd.SetHelpTemplate(helpTpl)
	cmd.SetUsageTemplate(usageTpl)
}

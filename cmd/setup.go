package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func missingToken(tokenVar string) {
	var (
		envTxt   string
		tokenURL string
	)
	if tokenVar == "linode" {
		tokenURL = "https://cloud.linode.com/profile/tokens"
		envTxt = "LINODE_TOKEN"
	} else {
		tokenURL = "https://app.pulumi.com/user/settings/tokens"
		envTxt = "PULUMI_ACCESS_TOKEN"
	}

	helpTxt := map[int]string{
		1: fmt.Sprintf("missing requirement: %s api token", tokenVar),
		2: "header:create a new token at:",
		3: "line:" + tokenURL,
		4: "header:set shell environment variable (~/.bashrc, ~/.bash_profile, etc.)",
		5: "header:example:",
		6: "line:" + fmt.Sprintf("`export %s=<TOKEN>`", envTxt),
		7: "line:`echo \"APL CLI\" >> ~/.bashrc`",
		8: "line:`echo \"export LINODE_TOKEN=<TOKEN>\" >> ~/.bashrc`",
	}

	for i := 1; i < 9; i++ {
		if i == 4 {
			fmt.Println() //nolint:forbidigo
		}

		logger.Warn(helpTxt[i])
	}

	fmt.Println() //nolint:forbidigo
}

func PreChk() {
	switch {
	case os.Getenv("LINODE_TOKEN") == "":
		missingToken("linode")
		logger.Error("linode api token: not found")
	case os.Getenv("PULUMI_ACCESS_TOKEN") == "":
		missingToken("pulumi")
		logger.Error("pulumi api token: not found")
	case os.Getuid() == 0:
		logger.Error("invalid user: do not run as root")
	}
}

func GetPulumiUser() string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	helpTxt := func() {
		txt := map[int]string{
			1: "missing requirement: pulumi",
			2: "header:install with:",
			3: "line:`curl -fsSL https://get.pulumi.com | sh`",
		}
		for i := 1; i < 4; i++ {
			logger.Warn(txt[i])
		}

		fmt.Println() //nolint:forbidigo
	}

	if err := os.Setenv("PULUMI_SKIP_UPDATE_CHECK", "true"); err != nil {
		logger.Error("skip update check: " + err.Error())
	}

	cmd := exec.CommandContext(ctx, "pulumi", "whoami")

	stdout, err := cmd.CombinedOutput()
	if err != nil {
		switch {
		case errors.Is(err, exec.ErrNotFound):
			helpTxt()
			logger.Error("pulumi login: pulumi not installed")
		default:
			logger.Error("pulumi login: " + err.Error())
		}
	}

	return string(stdout)
}

func SetupPrompt(promptStr string) string {
	var input string

	errStr := "input error: " + strings.ToLower(promptStr)
	prompt := promptStr + ": "
	regions := map[int]string{
		// APJ
		1: "ap-south",
		2: "au-mel",
		3: "id-cgk",
		4: "jp-osa",
		// EU
		5:  "es-mad",
		6:  "fr-par",
		7:  "gb-lon",
		8:  "it-mil",
		9:  "nl-ams",
		10: "se-sto",
		// Americas
		11: "br-gru",
		12: "us-east",
		13: "us-lax",
		14: "us-mia",
		15: "us-ord",
		16: "us-sea",
		17: "us-southeast",
	}
	stdin := bufio.NewReader(os.Stdin)

	switch promptStr {
	case "Region":
		logger.Info("header:APJ")

		for i := 1; i < 5; i++ {
			fstr := fmt.Sprintf("(%d)  %s", i, regions[i])
			logger.Info("line:" + fstr)
		}

		logger.Info("header:EU")

		for i := 5; i < 11; i++ {
			fstr := fmt.Sprintf("(%d)  %s", i, regions[i])
			logger.Info("line:" + fstr)
		}

		logger.Info("header:Americas")

		for i := 11; i < 18; i++ {
			fstr := fmt.Sprintf("(%d)  %s", i, regions[i])
			logger.Info("line:" + fstr)
		}

		logger.Info("input:" + prompt)

		if val, err := stdin.ReadString('\n'); err == nil {
			trimmed := strings.TrimSpace(val)

			n, _ := strconv.Atoi(trimmed)
			if _, ok := regions[n]; ok {
				input = regions[n]
			} else {
				logger.Error(errStr + "index out of range")
			}
		} else {
			logger.Error(errStr + err.Error())
		}
	default:
		logger.Info("input:" + prompt)

		val, err := stdin.ReadString('\n')
		if err != nil {
			logger.Error(errStr + err.Error())
		}

		input = val
	}

	return strings.TrimSpace(input)
}

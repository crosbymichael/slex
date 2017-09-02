package main

import (
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// SSHClientOptions holds the client options for establishing SSH connection.
// See 'man 5 ssh_config' for the option details.
type SSHClientOptions struct {
	ForwardAgent string
	Host         string
	HostName     string
	IdentityFile string
	Port         string
	ProxyCommand string
	User         string
}

// ParseSSHConfigFile parses the ~/.ssh/config file and build a list of sections.
func ParseSSHConfigFile() (map[string]SSHClientOptions, error) {
	sections := make(map[string]SSHClientOptions)

	// Read config file from default location ~/.ssh/config:
	user, err := user.Current()
	if err != nil {
		return sections, err
	}
	conf := filepath.Join(user.HomeDir, ".ssh", "config")

	log.Debugf("parsing ssh config file: %s", conf)
	content, err := ioutil.ReadFile(conf)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debugf("cannot find ssh config file: %s", conf)
			return sections, nil
		}
		return nil, err
	}

	// Read lines in reverse order and parse option for each Host section:
	lines := strings.Split(string(content), "\n")
	hostExpr := regexp.MustCompile("\\s*Host\\s*=?\\s*(.+)")

	end := len(lines) - 1
	for i := end; i >= 0; i-- {
		text := lines[i]

		// Skip comment lines:
		if strings.HasPrefix(text, "#") {
			continue
		}

		// When 'Host' option is found, parse the options of from current line to end line:
		m := hostExpr.FindStringSubmatch(text)
		if len(m) == 2 {
			host := m[1]
			sections[host] = ParseOptions(lines[i:end])

			end = i - 1 // The next line will be the end of the next section as we're doing reverse iteration.
		}
	}

	return sections, nil
}

// ParseOptions converts a list of OpenSSH client options to SSHClientOptions.
// Each option in the given list is a keyword-argument pair which is
// either separated by whitespace or optional whitespace and exactly one '='.
func ParseOptions(plainOpts []string) SSHClientOptions {
	optionExpr := regexp.MustCompile("\\s*(\\w+)\\s*=?\\s*(.+)")

	options := SSHClientOptions{
		Host: "*",  // Set Host pattern to "*" as default.
		Port: "22", // Set Port to "22" as default.
	}
	for _, i := range plainOpts {
		m := optionExpr.FindStringSubmatch(i)
		key := m[1]
		value := m[2]

		switch strings.ToLower(key) {
		case "host":
			options.Host = value
		case "hostname":
			options.HostName = value
		case "user":
			options.User = value
		case "port":
			options.Port = value
		case "forwardagent":
			options.ForwardAgent = value
		case "identityfile":
			options.IdentityFile = value
		case "proxycommand":
			options.ProxyCommand = value
		}
	}

	log.Debugf("parsed SSH options: %v", options)
	return options
}

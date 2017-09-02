package main

import (
	"io/ioutil"
	"os"
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

// ParseSSHConfigFile parses the file on the given file path and build a list of sections of SSH client options.
func ParseSSHConfigFile(path string) (map[string]SSHClientOptions, error) {
	sections := make(map[string]SSHClientOptions)

	log.Debugf("Parsing ssh config file: %s", path)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debugf("cannot find ssh config file: %s", path)
			return sections, nil
		}
		return nil, err
	}

	// Read lines in reverse order and parse option for each Host section:
	lines := strings.Split(string(content), "\n")
	hostExpr := regexp.MustCompile("\\s*Host\\s*=?\\s*(.+)")

	end := len(lines)
	for i := end - 1; i >= 0; i-- {
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

			end = i // This line will be the end of the next section as we're doing reverse iteration.
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
		if len(m) != 3 {
			continue
		}
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

	log.Debugf("Parsed SSH options: %v", options)
	return options
}

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

type SSHConfigFileSection struct {
	Host         string
	ForwardAgent string
	User         string
	HostName     string
	Port         string
	IdentityFile string
}

// parseSSHConfigFile parses the ~/.ssh/config file and build a list of section
func parseSSHConfigFile() (map[string]*SSHConfigFileSection, error) {

	sections := make(map[string]*SSHConfigFileSection)

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
	lines := strings.Split(string(content), "\n")
	current := &SSHConfigFileSection{}
	for _, line := range lines {
		parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])
		if key == "host" {
			if current.Host != "" {
				sections[current.Host] = current
			}
			current = &SSHConfigFileSection{Host: val}
		} else if key == "hostname" {
			current.HostName = val
		} else if key == "user" {
			current.User = val
		} else if key == "port" {
			current.Port = val
		} else if key == "forwardagent" {
			current.ForwardAgent = val
		}
	}

	// add last host to map
	if current.Host != "" {
		sections[current.Host] = current
	}

	return sections, nil
}

// ParseOptions converts a list of OpenSSH client options to key-value pairs.
// Each option in the list is a keyword-argument pair which is
// either separated by whitespace or optional whitespace and exactly one '='.
// For the full list of options, see man 5 ssh_config.
func ParseOptions(plainOpts []string) map[string]string {
	optionExpr := regexp.MustCompile("(\\w+)\\s*=?\\s*(.+)")
	options := make(map[string]string)

	for _, i := range plainOpts {
		m := optionExpr.FindStringSubmatch(i)
		options[m[1]] = m[2]
	}

	log.Debugf("parsed SSH options: %v", options)
	return options
}

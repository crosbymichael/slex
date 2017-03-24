package main

import (
	"io/ioutil"
	"os"
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

// parseSSHConfigFileSection parses a section from the ~/.ssh/config file
func parseSSHConfigFileSection(content string) *SSHConfigFileSection {
	section := &SSHConfigFileSection{}

	for n, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if n == 0 {
			section.Host = line
		} else if strings.HasPrefix(line, "ForwardAgent") {
			section.ForwardAgent = strings.TrimSpace(strings.TrimPrefix(line, "ForwardAgent"))
		} else if strings.HasPrefix(line, "User") {
			section.User = strings.TrimSpace(strings.TrimPrefix(line, "User"))
		} else if strings.HasPrefix(line, "HostName") {
			section.HostName = strings.TrimSpace(strings.TrimPrefix(line, "HostName"))
		} else if strings.HasPrefix(line, "Port") {
			section.Port = strings.TrimSpace(strings.TrimPrefix(line, "Port"))
		} else if strings.HasPrefix(line, "IdentityFile") {
			section.IdentityFile = strings.TrimSpace(strings.TrimPrefix(line, "IdentityFile"))
		}
	}
	log.Debugf("parsed ssh config file section: %s", section.Host)
	return section
}

// parseSSHConfigFile parses the ~/.ssh/config file and build a list of section
func parseSSHConfigFile(path string) (map[string]*SSHConfigFileSection, error) {

	sections := make(map[string]*SSHConfigFileSection)

	log.Debugf("parsing ssh config file: %s", path)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debugf("cannot find ssh config file: %s", path)
			return sections, nil
		}
		return nil, err
	}

	for _, split := range strings.Split(string(content), "Host ") {
		split = strings.TrimSpace(split)
		if split == "" {
			continue
		}

		section := parseSSHConfigFileSection(split)
		sections[section.Host] = section
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

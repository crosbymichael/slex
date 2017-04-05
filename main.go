package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// preload initializes any global options and configuration
// before the main or sub commands are run
func preload(context *cli.Context) error {
	if context.GlobalBool("debug") {
		log.SetLevel(log.DebugLevel)
	}
	return nil
}

// loadHosts returns a list of host addresses that are specified on the
// command line and also in a hosts file separated by new lines.
func loadHosts(context *cli.Context) ([]string, error) {
	hosts := []string(context.GlobalStringSlice("host"))
	if hostsFile := context.GlobalString("hosts"); hostsFile != "" {
		f, err := os.Open(hostsFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		s := bufio.NewScanner(f)
		for s.Scan() {
			hosts = append(hosts, s.Text())
		}
		if err := s.Err(); err != nil {
			return nil, err
		}
	}
	return hosts, nil
}

// multiplexAction uses the arguments passed via the command line and
// multiplexes them across multiple SSH connections
func multiplexAction(context *cli.Context) error {
	c, err := newCommand(context)
	if err != nil {
		return err
	}
	log.Debug(c)

	hosts, err := loadHosts(context)
	if err != nil {
		return err
	}

	sections, err := parseSSHConfigFile()
	if err != nil {
		return err
	}

	if len(hosts) == 0 {
		return fmt.Errorf("no host specified for command to run")
	}
	log.Debugf("hosts %v", hosts)

	agentForwarding := context.GlobalBool("A")
	var agt agent.Agent
	if agentForwarding {
		agt, err = newAgent()
		if err != nil {
			return err
		}
	}

	identityFiles := []string{}
	if c.Identity != "" {
		identityFiles = append(identityFiles, c.Identity)
	}
	methods := defaultAuthMethods(identityFiles, agt)

	plainOptions := []string(context.GlobalStringSlice("option"))
	options := ParseOptions(plainOptions)

	quiet := context.GlobalBool("quiet")
	group := &sync.WaitGroup{}
	for _, h := range hosts {
		group.Add(1)
		go executeCommand(c, h, sections[h], agt, methods, quiet, options, group)
	}
	group.Wait()
	log.Debugf("finished executing %s on all hosts", c)
	return nil
}

func executeCommand(c command, host string, section *SSHConfigFileSection, agt agent.Agent, methods map[string]ssh.AuthMethod, quiet bool, options map[string]string, group *sync.WaitGroup) {
	defer group.Done()
	var (
		err          error
		originalHost = host
	)
	if host, err = cleanHost(host); err != nil {
		log.WithField("host", originalHost).Error(err)
		return
	}

	if err = runSSH(c, host, section, agt, methods, quiet, options); err != nil {
		log.WithField("host", host).Error(err)
		return
	}
}

// runSSH executes the given command on the given host.
// All available SSH authentication methods to the host will be tried.
func runSSH(c command, host string, section *SSHConfigFileSection, agt agent.Agent, methods map[string]ssh.AuthMethod, quiet bool, options map[string]string) error {
	user := c.User

	// Update SSH parameters from ~/.ssh/config
	if section != nil {
		log.Debugf("Updating SSH client parameters for host: %s", host)
		updateFromSSHConfigFile(section, &host, &user, &methods)
	}

	// Try using each available AuthMethod to establish SSH session
	var (
		session *sshSession
		err     error
	)
	for k, m := range methods {
		config := newSSHClientConfig(user, host, section, agt, m)
		session, err = config.NewSession(options)
		if err == nil {
			log.Debugf("Session established using identity file %s", k)
			break // Session established, quit trying the next AuthMethod
		}

		log.Debugf("Failed to establish session using identity file %s - %v", k, err)
	}

	if session == nil {
		return fmt.Errorf("none of the provided authentication methods can establish SSH session successfully")
	}

	if !quiet {
		w := newBufCloser(os.Stdout)
		defer w.Close()
		session.Stderr, session.Stdout = w, w
	}
	defer func() {
		session.Close()
		log.Printf("Session complete from %s@%s", user, host)
	}()

	for key, value := range c.Env {
		if err := session.Setenv(key, value); err != nil {
			return err
		}
	}
	return session.Run(c.Cmd)
}

// cleanHost parses out the hostname/ip and port.  If no port is
// specified then port 22 is appended to the hostname/ip
func cleanHost(host string) (string, error) {
	h, port, err := net.SplitHostPort(host)
	if err != nil {
		if !strings.Contains(err.Error(), "missing port in address") {
			return "", err
		}
		port = "22"
		h = host
	}
	if port == "" {
		port = "22"
	}
	return net.JoinHostPort(h, port), nil
}

func main() {
	app := cli.NewApp()
	app.Name = "slex"
	app.Usage = "SSH commands multiplexed"
	app.Version = "2"
	app.Author = "@crosbymichael"
	app.Email = "crosbymichael@gmail.com"
	app.Before = preload
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "enable debug output for the logs",
		},
		cli.StringSliceFlag{
			Name:  "host",
			Value: &cli.StringSlice{},
			Usage: "SSH host address",
		},
		cli.StringFlag{
			Name:  "hosts",
			Usage: "file containing host addresses separated by a new line",
		},
		cli.StringFlag{
			Name:  "user,u",
			Value: "root",
			Usage: "user to execute the command as",
		},
		cli.StringFlag{
			Name:  "identity,i",
			Usage: "SSH identity to use for connecting to the host",
		},
		cli.StringSliceFlag{
			Name:  "option,o",
			Value: &cli.StringSlice{},
			Usage: "SSH client option",
		},
		cli.BoolFlag{
			Name:  "agent,A",
			Usage: "Forward authentication request to the ssh agent",
		},
		cli.StringSliceFlag{
			Name:  "env,e",
			Usage: "set environment variables for SSH command",
			Value: &cli.StringSlice{},
		},
		cli.BoolFlag{
			Name:  "quiet,q",
			Usage: "disable output from the ssh command",
		},
	}
	app.Action = multiplexAction
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

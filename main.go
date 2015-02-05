package main

import (
	"bufio"
	"net"
	"os"
	"strings"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

// preload initializes any global options and configuration
// before the main or sub commands are run
func preload(context *cli.Context) error {
	if context.GlobalBool("debug") {
		log.SetLevel(log.DebugLevel)
	}
	return nil
}

// hostHosts returns a list of host addresses that are specified on the
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
			if err := s.Err(); err != nil {
				return nil, err
			}
			hosts = append(hosts, s.Text())
		}
	}
	return hosts, nil
}

// multiplexAction uses the arguments passed via the command line and
// multiplexes them across multiple SSH connections
func multiplexAction(context *cli.Context) {
	c, err := newCommand(context)
	if err != nil {
		log.Fatal(err)
	}
	log.Debug(c)

	hosts, err := loadHosts(context)
	if err != nil {
		log.Fatal(err)
	}
	if len(hosts) == 0 {
		log.Fatal("no host specified for command to run")
	}
	log.Debugf("hosts %v", hosts)
	group := &sync.WaitGroup{}
	for _, h := range hosts {
		group.Add(1)
		go executeCommand(c, h, context.GlobalBool("A"), context.GlobalBool("quiet"), group)
	}
	group.Wait()
	log.Debugf("finished executing %s on all hosts", c)
}

func executeCommand(c command, host string, agentForwarding, quiet bool, group *sync.WaitGroup) {
	defer group.Done()
	var (
		err          error
		originalHost = host
	)

	if host, err = cleanHost(host); err != nil {
		log.WithField("host", originalHost).Error(err)
		return
	}

	if err = runSSH(c, host, agentForwarding, quiet); err != nil {
		log.WithField("host", host).Error(err)
		return
	}
	log.Debugf("host %s executed successfully", host)
}

// runSSH executes the given command on the given host
func runSSH(c command, host string, agentForwarding, quiet bool) error {
	config, err := newSshClientConfig(c.User, c.Identity, agentForwarding)
	if err != nil {
		return err
	}
	session, err := config.NewSession(host)
	if err != nil {
		return err
	}
	defer session.Close()

	// TODO: find a better way to multiplex all the streams
	// and support STDIN without sending to all sessions
	if !quiet {
		session.Stderr = newNameWriter(host, os.Stderr)
		session.Stdout = newNameWriter(host, os.Stdout)
	}
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
	app.Version = "1"
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
			Value: "id_rsa",
			Usage: "SSH identity to use for connecting to the host",
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

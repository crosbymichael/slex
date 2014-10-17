package main

import (
	"net"
	"os"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

var (
	logger = logrus.New()

	globalFlags = []cli.Flag{
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
	}
)

// preload initializes any global options and configuration
// before the main or sub commands are run
func preload(context *cli.Context) error {
	if context.GlobalBool("debug") {
		logger.Level = logrus.DebugLevel
	}
	return nil
}

// multiplexAction uses the arguments passed via the command line and
// multiplexes them across multiple SSH connections
func multiplexAction(context *cli.Context) {
	c, err := newCommand(context)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Debug(c)

	hosts := []string(context.GlobalStringSlice("host"))
	if len(hosts) == 0 {
		logger.Fatal("no host specified for command to run")
	}
	logger.Debugf("hosts %v", hosts)

	af := context.GlobalBool("A")

	group := &sync.WaitGroup{}
	for _, h := range hosts {
		group.Add(1)
		go executeCommand(c, h, af, group)
	}

	group.Wait()

	logger.Debugf("finished executing %s on all hosts", c)
}

func executeCommand(c command, host string, agentForwarding bool, group *sync.WaitGroup) {
	defer group.Done()
	var (
		err          error
		originalHost = host
	)

	if host, err = cleanHost(host); err != nil {
		logger.WithField("host", originalHost).Error(err)
		return
	}

	if err = runSSH(c, host, agentForwarding); err != nil {
		logger.WithField("host", host).Error(err)
		return
	}
	logger.Debugf("host %s executed successfully", host)
}

// runSSH executes the given command on the given host
func runSSH(c command, host string, agentForwarding bool) error {
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
	session.Stderr = newNameWriter(host, os.Stderr)
	session.Stdout = newNameWriter(host, os.Stdout)

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
	app.Flags = globalFlags
	app.Action = multiplexAction

	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err)
	}
}

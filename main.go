package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"code.google.com/p/go.crypto/ssh"

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
	}
)

type command struct {
	User     string
	Args     []string
	Identity string
}

func (c command) cmd() string {
	return strings.Join(c.Args, " ")
}

func (c command) config() (*ssh.ClientConfig, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	contents, err := ioutil.ReadFile(filepath.Join(u.HomeDir, ".ssh", c.Identity))
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(contents)
	if err != nil {
		return nil, err
	}
	return &ssh.ClientConfig{
		User: c.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}, nil
}

func (c command) String() string {
	return fmt.Sprintf("user: %s command: %v", c.User, c.Args)
}

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
	c, err := createCommand(context)
	if err != nil {
		logger.Fatal(err)
	}
	logger.Debugf("command %s", c)

	hosts := []string(context.GlobalStringSlice("host"))
	if len(hosts) == 0 {
		logger.Fatal("no host specified for command to run")
	}
	logger.Debugf("hosts %v", hosts)

	logger.Infof("executing command on %d hosts", len(hosts))

	group := &sync.WaitGroup{}
	for _, h := range hosts {
		group.Add(1)

		go executeCommand(c, h, group)
	}

	group.Wait()
	logger.Infof("finished executing %s on all hosts", c)
}

func createCommand(context *cli.Context) (c command, err error) {
	args := []string(context.Args())
	if len(args) == 0 {
		return c, fmt.Errorf("no command specified to execute")
	}
	return command{
		User:     context.GlobalString("user"),
		Args:     args,
		Identity: context.GlobalString("identity"),
	}, nil
}

func executeCommand(c command, host string, group *sync.WaitGroup) {
	defer group.Done()
	var (
		err          error
		originalHost = host
	)

	if host, err = cleanHost(host); err != nil {
		logger.WithField("host", originalHost).Error(err)
		return
	}

	if err = runSSH(c, host); err != nil {
		logger.WithField("host", host).Error(err)
		return
	}
	logger.Infof("host %s executed successfully", host)
}

func runSSH(c command, host string) error {
	config, err := c.config()
	if err != nil {
		return err
	}
	conn, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return err
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	// TODO: find a better way to multiplex all the streams
	// and support STDIN without sending to all sessions
	session.Stderr = os.Stderr
	session.Stdout = os.Stdout

	return session.Run(c.cmd())
}

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

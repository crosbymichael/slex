package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/codegangsta/cli"

	"code.google.com/p/go.crypto/ssh"
)

// newCommand returns a command populated from the context
func newCommand(context *cli.Context) (c command, err error) {
	var (
		cmd  string
		args = []string(context.Args())
	)

	switch len(args) {
	default:
		cmd = strings.Join(args, " ")
	case 0:
		raw, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			return c, err
		}
		cmd = string(raw)
	}

	if cmd == "" {
		return c, fmt.Errorf("no command specified")
	}
	return command{
		Cmd:      cmd,
		User:     context.GlobalString("user"),
		Identity: context.GlobalString("identity"),
	}, nil
}

// command to run over an SSH connection
type command struct {
	// User is the user to run the command as
	User string

	// Cmd is the pared command string that will be executed
	Cmd string

	// Identity is the SSH key to identify as which is commonly
	// the private keypair i.e. id_rsa
	Identity string
}

// config returns the SSH client config for the connection
func (c command) config() (*ssh.ClientConfig, error) {
	contents, err := c.loadIdentity()
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

// loadIdentity returns the private key file's contents
func (c command) loadIdentity() ([]byte, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}
	return ioutil.ReadFile(filepath.Join(u.HomeDir, ".ssh", c.Identity))
}

// String returns a pretty printed string of the command
func (c command) String() string {
	return fmt.Sprintf("user: %s command: %s", c.User, c.Cmd)
}

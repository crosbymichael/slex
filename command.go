package main

import (
	"fmt"
	"io/ioutil"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/codegangsta/cli"

	"code.google.com/p/go.crypto/ssh"
)

// newCommand returns a command populated from the context
func newCommand(context *cli.Context) (c command, err error) {
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

// command to run over an SSH connection
type command struct {
	// User is the user to run the command as
	User string

	// Args are the CLI arguments to execute
	Args []string

	// Identity is the SSH key to identify as which is commonly
	// the private keypair i.e. id_rsa
	Identity string
}

// cmd returns the command's args joined by " "
func (c command) cmd() string {
	return strings.Join(c.Args, " ")
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
	return fmt.Sprintf("user: %s command: %v", c.User, c.Args)
}

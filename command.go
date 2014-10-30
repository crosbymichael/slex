package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"strings"

	"github.com/codegangsta/cli"
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
	env, err := parseEnvironment(context)
	if err != nil {
		return c, err
	}
	return command{
		Cmd:      cmd,
		User:     context.GlobalString("user"),
		Identity: context.GlobalString("identity"),
		Env:      env,
	}, nil
}

func parseEnvironment(context *cli.Context) (map[string]string, error) {
	env := make(map[string]string)
	for _, v := range context.GlobalStringSlice("env") {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid env format %s", v)
		}
		env[parts[0]] = parts[1]
	}
	return env, nil
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

	// Env are environment variables to pass to the SSH command
	Env map[string]string
}

// String returns a pretty printed string of the command
func (c command) String() string {
	return fmt.Sprintf("user: %s command: %s", c.User, c.Cmd)
}

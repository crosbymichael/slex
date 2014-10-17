package main

import (
	"errors"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"

	"code.google.com/p/go.crypto/ssh"
	"code.google.com/p/go.crypto/ssh/agent"
)

// sshClientConfig stores the configuration
// and the ssh agent to forward authentication requests
type sshClientConfig struct {
	// agent is the connection to the ssh agent
	agent agent.Agent

	*ssh.ClientConfig
}

// newSshClientConfig initializes the ssh configuration.
// It connects with the ssh agent when agent forwarding is enabled.
func newSshClientConfig(userName, identity string, agentForwarding bool) (*sshClientConfig, error) {
	if agentForwarding {
		return newSshAgentConfig(userName)
	}

	return newSshDefaultConfig(userName, identity)
}

func newSshAgentConfig(userName string) (*sshClientConfig, error) {
	agent, err := newAgent()
	if err != nil {
		return nil, err
	}

	config, err := sshAgentConfig(userName, agent)
	if err != nil {
		return nil, err
	}

	return &sshClientConfig{
		agent:        agent,
		ClientConfig: config,
	}, nil
}

func newSshDefaultConfig(userName, identity string) (*sshClientConfig, error) {
	config, err := sshDefaultConfig(userName, identity)
	if err != nil {
		return nil, err
	}

	return &sshClientConfig{ClientConfig: config}, nil
}

// NewSession creates a new ssh session with the host.
// It forwards authentication to the agent when it's configured.
func (s *sshClientConfig) NewSession(host string) (*ssh.Session, error) {
	conn, err := ssh.Dial("tcp", host, s.ClientConfig)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if s.agent != nil {
		if err := agent.ForwardToAgent(conn, s.agent); err != nil {
			return nil, err
		}
	}

	session, err := conn.NewSession()
	if s.agent != nil {
		err = agent.RequestAgentForwarding(session)
	}

	return session, err
}

// newAgent connects with the SSH agent in the to forward authentication requests.
func newAgent() (agent.Agent, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, errors.New("Unable to connect to the ssh agent. Please, check that SSH_AUTH_SOCK is set and the ssh agent is running")
	}

	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, err
	}

	return agent.NewClient(conn), nil
}

func sshAgentConfig(userName string, a agent.Agent) (*ssh.ClientConfig, error) {
	signers, err := a.Signers()
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signers...),
		},
	}, nil
}

// sshDefaultConfig returns the SSH client config for the connection
func sshDefaultConfig(userName, identity string) (*ssh.ClientConfig, error) {
	contents, err := loadDefaultIdentity(userName, identity)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(contents)
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}, nil
}

// loadDefaultIdentity returns the private key file's contents
func loadDefaultIdentity(userName, identity string) ([]byte, error) {
	u, err := user.Current()
	if err != nil {
		return nil, err
	}

	return ioutil.ReadFile(filepath.Join(u.HomeDir, ".ssh", identity))
}

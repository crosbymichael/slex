package main

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
)

// sshSession stores the open session and connection to execute a command.
type sshSession struct {
	// conn is the ssh client that started the session.
	conn *ssh.Client

	*ssh.Session
}

// Close closses the open ssh session and connection.
func (s *sshSession) Close() {
	s.Session.Close()
	s.conn.Close()
}

// sshClientConfig stores the configuration
// and the ssh agent to forward authentication requests
type sshClientConfig struct {
	// agent is the connection to the ssh agent
	agent agent.Agent

	// host to connect to
	host string

	*ssh.ClientConfig
}

// updateFromSSHConfigFile updates the host, username and agentforwarding parameters
// from the ~/.ssh/config if there is a matching section
func updateFromSSHConfigFile(section *SSHConfigFileSection, host, userName *string, agentForwarding *bool) {
	hostName, port, err := net.SplitHostPort(*host)
	if err != nil {
		return
	}

	if section.ForwardAgent == "yes" {
		*agentForwarding = true
	} else if section.ForwardAgent == "no" {
		*agentForwarding = false
	}
	if section.User != "" {
		*userName = section.User
	}
	if section.HostName != "" {
		hostName = section.HostName
	}
	if section.Port != "" {
		port = section.Port
	}
	*host = net.JoinHostPort(hostName, port)
}

// newSSHClientConfig initializes the ssh configuration.
// It connects with the ssh agent when agent forwarding is enabled.
func newSSHClientConfig(host string, section *SSHConfigFileSection, userName, identity string, agentForwarding bool) (*sshClientConfig, error) {
	var (
		config *sshClientConfig
		err    error
	)

	if section != nil {
		updateFromSSHConfigFile(section, &host, &userName, &agentForwarding)
	}

	if agentForwarding {
		config, err = newSSHAgentConfig(userName)
	} else {
		config, err = newSSHDefaultConfig(userName, identity)
	}

	if config != nil {
		config.host = host
	}
	return config, err
}

// newSSHAgentConfig initializes the configuration to talk with an ssh agent.
func newSSHAgentConfig(userName string) (*sshClientConfig, error) {
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

// newSSHDefaultConfig initializes the configuration to use an ideitity file.
func newSSHDefaultConfig(userName, identity string) (*sshClientConfig, error) {
	config, err := sshDefaultConfig(userName, identity)
	if err != nil {
		return nil, err
	}

	return &sshClientConfig{ClientConfig: config}, nil
}

// NewSession creates a new ssh session with the host.
// It forwards authentication to the agent when it's configured.
func (s *sshClientConfig) NewSession(host string) (*sshSession, error) {
	conn, err := ssh.Dial("tcp", host, s.ClientConfig)
	if err != nil {
		return nil, err
	}

	if s.agent != nil {
		if err := agent.ForwardToAgent(conn, s.agent); err != nil {
			return nil, err
		}
	}

	session, err := conn.NewSession()
	if s.agent != nil {
		err = agent.RequestAgentForwarding(session)
	}

	return &sshSession{
		conn:    conn,
		Session: session,
	}, err
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

// sshAgentConfig creates a new configuration for the ssh client
// with the signatures from the ssh agent.
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
	contents, err := loadIdentity(userName, identity)
	if err != nil {
		return nil, err
	}

	// handle plain and encrypted private key file
	block, _ := pem.Decode(contents)
	if block == nil {
		return nil, fmt.Errorf("cannot decode private key file %s", identity)
	}

	var signer ssh.Signer
	if x509.IsEncryptedPEMBlock(block) {
		fmt.Print("Key passphrase: ")
		pass, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return nil, err
		}
		block.Bytes, err = x509.DecryptPEMBlock(block, pass)
		if err != nil {
			return nil, err
		}

		var key interface{}
		switch block.Type {
		case "RSA PRIVATE KEY":
			key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		case "EC PRIVATE KEY":
			key, err = x509.ParseECPrivateKey(block.Bytes)
		case "DSA PRIVATE KEY":
			key, err = ssh.ParseDSAPrivateKey(block.Bytes)
		default:
			return nil, fmt.Errorf("unsupported key type %q", block.Type)
		}
		if err != nil {
			return nil, err
		}

		signer, err = ssh.NewSignerFromKey(key)
		if err != nil {
			return nil, err
		}
	} else {
		signer, err = ssh.ParsePrivateKey(contents)
		if err != nil {
			return nil, err
		}
	}

	return &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}, nil
}

// loadIdentity returns the private key file's contents
func loadIdentity(userName, identity string) ([]byte, error) {
	if filepath.Dir(identity) == "." {
		u, err := user.Current()
		if err != nil {
			return nil, err
		}
		identity = filepath.Join(u.HomeDir, ".ssh", identity)
	}

	return ioutil.ReadFile(identity)
}

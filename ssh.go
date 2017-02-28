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
	"strconv"
	"syscall"
	"time"

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

// updateFromSSHConfigFile updates SSH client parameters
// from the ~/.ssh/config if there is a matching section.
func updateFromSSHConfigFile(section *SSHConfigFileSection, host, user *string, methods *map[string]ssh.AuthMethod) {
	if section.User != "" {
		*user = section.User
	}

	if section.HostName != "" && section.Port != "" {
		*host = net.JoinHostPort(section.HostName, section.Port)
	}

	if section.IdentityFile != "" {
		if identity, err := resolveIdentity(section.IdentityFile); err == nil {
			if m, err := newSSHPublicKeyAuthMethod(*user, identity); err == nil {
				(*methods)[identity] = m
			}
		}
	}
}

// newSSHClientConfig initializes per-host SSH configuration.
func newSSHClientConfig(user, host string, section *SSHConfigFileSection, agt agent.Agent, method ssh.AuthMethod, options map[string]string) (*sshClientConfig, error) {
	var timeout time.Duration

	if ct, ok := options["ConnectTimeout"]; ok {
		t, err := strconv.Atoi(ct)
		if err != nil {
			return nil, fmt.Errorf("ConnectTimeout is not an integer")
		}
		timeout = time.Second * time.Duration(t)
	}

	config := &ssh.ClientConfig{
		User:    user,
		Auth:    []ssh.AuthMethod{method},
		Timeout: timeout,
	}
	return &sshClientConfig{
		agent:        agt,
		host:         host,
		ClientConfig: config,
	}, nil
}

// NewSession creates a new ssh session with the host.
// It forwards authentication to the agent when it's configured.
func (s *sshClientConfig) NewSession(options map[string]string) (*sshSession, error) {
	var (
		conn *ssh.Client
		err  error
	)

	if proxyCmd, ok := options["ProxyCommand"]; ok {
		cmdConn, err := NewProxyCmdConn(s, proxyCmd)
		if err != nil {
			return nil, err
		}
		c, chans, reqs, err := ssh.NewClientConn(cmdConn, "", s.ClientConfig)
		if err != nil {
			return nil, err
		}
		conn = ssh.NewClient(c, chans, reqs)
	} else {
		conn, err = ssh.Dial("tcp", s.host, s.ClientConfig)
		if err != nil {
			return nil, err
		}
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

// defaultAuthMethods initializes all the available SSH authentication method.
func defaultAuthMethods(user, identity string, agt agent.Agent) map[string]ssh.AuthMethod {
	methods := make(map[string]ssh.AuthMethod)

	if m, err := newSSHPublicKeyAuthMethod(user, identity); err == nil {
		methods[identity] = m
	}

	if agt != nil {
		if m, err := newSSHAgentAuthMethod(agt); err == nil {
			methods["ssh-agent"] = m
		}
	}

	return methods
}

// newSSHPublicKeyAuthMethod creates a new SSH authentication method using SSH agent
func newSSHAgentAuthMethod(agt agent.Agent) (ssh.AuthMethod, error) {
	signers, err := agt.Signers()
	if err != nil {
		return nil, err
	}

	return ssh.PublicKeys(signers...), nil
}

// newSSHPublicKeyAuthMethod creates a new SSH authentication method using public/private key
func newSSHPublicKeyAuthMethod(userName, identity string) (ssh.AuthMethod, error) {
	contents, err := ioutil.ReadFile(identity)
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
		fmt.Println("")

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

	return ssh.PublicKeys(signer), nil
}

// resolveIdentity returns the realpath of the private key file.
func resolveIdentity(identity string) (string, error) {
	if filepath.Dir(identity) != "." {
		return identity, nil
	}

	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(u.HomeDir, ".ssh", identity), nil
}

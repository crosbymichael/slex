package main

import (
	"io"
	"net"
	"os/exec"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	shlex "github.com/flynn/go-shlex"
)

// ProxyCmdConn is a Conn for talking to the underlying ProxyCommand.
type ProxyCmdConn struct {
	io.ReadCloser
	io.WriteCloser
}

// NewProxyCmdConn creates a new ProxyCmdConn
// and starts the underlying ProxyCommand.
func NewProxyCmdConn(s *sshClientConfig, cmd string) (*ProxyCmdConn, error) {
	host, port, err := net.SplitHostPort(s.host)
	if err != nil {
		return nil, err
	}

	cmd = strings.Replace(cmd, "%h", host, -1)
	cmd = strings.Replace(cmd, "%p", port, -1)
	args, err := shlex.Split(cmd)
	if err != nil {
		return nil, err
	}

	c := exec.Command(args[0], args[1:]...)

	stdin, err := c.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, err
	}

	// FIXME: Report errors from StderrPipe
	if err := c.Start(); err != nil {
		return nil, err
	}
	log.Debugf("ProxyCommand started: '%s %s'.", args[0], strings.Join(args[1:], " "))

	return &ProxyCmdConn{
		ReadCloser:  stdout,
		WriteCloser: stdin,
	}, nil
}

func (c *ProxyCmdConn) Close() error {
	// Stdin pipe must be closed before stdout pipe
	// so that the underlying command knows it's time to end.
	// Otherwise, closing the stdout pipe first will be blocked forever.
	if err := c.WriteCloser.Close(); err != nil {
		return err
	}
	if err := c.ReadCloser.Close(); err != nil {
		return err
	}

	return nil
}

func (c *ProxyCmdConn) LocalAddr() net.Addr {
	return nil
}

func (c *ProxyCmdConn) RemoteAddr() net.Addr {
	return nil
}

func (c *ProxyCmdConn) SetDeadline(t time.Time) error {
	// FIXME: Implement timeout
	return nil
}

func (c *ProxyCmdConn) SetReadDeadline(t time.Time) error {
	// FIXME: Implement timeout
	return nil
}

func (c *ProxyCmdConn) SetWriteDeadline(t time.Time) error {
	// FIXME: Implement timeout
	return nil
}

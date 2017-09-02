package main

import (
	"io/ioutil"
	"syscall"
	"testing"
)

func TestParseOptions(t *testing.T) {
	verify := func(fmt string, exp, out SSHClientOptions) {
		if exp != out {
			t.Errorf("Could not parse option - format: %s, expected: %q, output: %q.", fmt, exp, out)
		}
	}

	// Test option pattern '<keyword> <argument>'
	{
		fmt := "<keyword> <argument>"
		in := []string{"Host 127.0.0.1", "Port 22"}
		exp := SSHClientOptions{
			Host: "127.0.0.1",
			Port: "22",
		}
		out := ParseOptions(in)

		verify(fmt, exp, out)
	}

	// Test option pattern '<keyword>=<argument>'
	{
		fmt := "<keyword>=<argument>"
		in := []string{"Host=127.0.0.1", "Port=22"}
		exp := SSHClientOptions{
			Host: "127.0.0.1",
			Port: "22",
		}
		out := ParseOptions(in)

		verify(fmt, exp, out)
	}

	// Test option pattern '<keyword> = <argument>'
	{
		fmt := "<keyword> = <argument>"
		in := []string{"Host = 127.0.0.1", "Port = 22"}
		exp := SSHClientOptions{
			Host: "127.0.0.1",
			Port: "22",
		}
		out := ParseOptions(in)

		verify(fmt, exp, out)
	}

	// Test option pattern '<keyword><tab><argument>'
	{
		fmt := "<keyword><tab><argument>"
		in := []string{"Host	127.0.0.1", "Port	22"}
		exp := SSHClientOptions{
			Host: "127.0.0.1",
			Port: "22",
		}
		out := ParseOptions(in)

		verify(fmt, exp, out)
	}
}

func TestParseSSHConfigFile(t *testing.T) {
	verify := func(content string, exp, out map[string]SSHClientOptions) {
		if len(exp) != len(out) {
			t.Errorf("Some sections are not parsed - content: %q, expected: %q, output: %q.", content, exp, out)
		}

		for k, e := range exp {
			o, ok := out[k]
			if !ok {
				t.Errorf("Section is missing - content: %q, expected: '%s', output: '%q'.", content, k, o)
			} else if o != e {
				t.Errorf("Could not parse section - content: %q, expected: '%s: %q', output: '%s: %q'.", content, k, e, k, o)
			}
		}
	}

	f, err := ioutil.TempFile("", "test-ssh-conf")
	if err != nil {
		panic(err)
	}
	defer syscall.Unlink(f.Name())

	// Test blank options file
	{
		in := "# <blank>"
		ioutil.WriteFile(f.Name(), []byte(in), 0644)
		exp := map[string]SSHClientOptions{}
		out, _ := ParseSSHConfigFile(f.Name())

		verify(in, exp, out)
	}

	// Test options file with 1 section
	{
		in := `
Host github.com
  User github
`
		ioutil.WriteFile(f.Name(), []byte(in), 0644)
		exp := map[string]SSHClientOptions{}
		exp["github.com"] = SSHClientOptions{
			Host: "github.com",
			Port: "22",
			User: "github",
		}
		out, _ := ParseSSHConfigFile(f.Name())

		verify(in, exp, out)
	}

	// Test options file with 2 sections
	{
		in := `
Host github.com
  User github

Host bitbucket.com
  User bitbucket
`
		ioutil.WriteFile(f.Name(), []byte(in), 0644)
		exp := map[string]SSHClientOptions{}
		exp["github.com"] = SSHClientOptions{
			Host: "github.com",
			Port: "22",
			User: "github",
		}
		exp["bitbucket.com"] = SSHClientOptions{
			Host: "bitbucket.com",
			Port: "22",
			User: "bitbucket",
		}
		out, _ := ParseSSHConfigFile(f.Name())

		verify(in, exp, out)
	}

	// Test options file with blank sections
	{
		in := `
Host github.com
Host bitbucket.com
`
		ioutil.WriteFile(f.Name(), []byte(in), 0644)
		exp := map[string]SSHClientOptions{}
		exp["github.com"] = SSHClientOptions{
			Host: "github.com",
			Port: "22",
		}
		exp["bitbucket.com"] = SSHClientOptions{
			Host: "bitbucket.com",
			Port: "22",
		}
		out, _ := ParseSSHConfigFile(f.Name())

		verify(in, exp, out)
	}
}

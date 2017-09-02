package main

import (
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

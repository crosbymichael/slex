package main

import (
	"testing"
)

func TestParseOptions(t *testing.T) {
	verify := func(fmt string, exp map[string]string, out map[string]string) {
		if len(out) != len(exp) {
			t.Errorf("Some options are not parsed - format: %s, expected: %q, output: %q.", fmt, exp, out)
		}
		for k, v := range exp {
			o := out[k]
			if o != v {
				t.Errorf("Could not parse option - format: %s, expected: '%s: %s', output: '%s: %s'.", fmt, k, v, k, o)
			}
		}
	}

	// Test option pattern '<keyword> <argument>'
	{
		fmt := "<keyword> <argument>"
		in := []string{"Host 127.0.0.1", "Port 22"}
		exp := map[string]string{"Host": "127.0.0.1", "Port": "22"}
		out := ParseOptions(in)

		verify(fmt, exp, out)
	}

	// Test option pattern '<keyword>=<argument>'
	{
		fmt := "<keyword>=<argument>"
		in := []string{"Host=127.0.0.1", "Port=22"}
		exp := map[string]string{"Host": "127.0.0.1", "Port": "22"}
		out := ParseOptions(in)

		verify(fmt, exp, out)
	}

	// Test option pattern '<keyword> = <argument>'
	{
		fmt := "<keyword> = <argument>"
		in := []string{"Host = 127.0.0.1", "Port = 22"}
		exp := map[string]string{"Host": "127.0.0.1", "Port": "22"}
		out := ParseOptions(in)

		verify(fmt, exp, out)
	}

	// Test option pattern '<keyword><tab><argument>'
	{
		fmt := "<keyword><tab><argument>"
		in := []string{"Host	127.0.0.1", "Port	22"}
		exp := map[string]string{"Host": "127.0.0.1", "Port": "22"}
		out := ParseOptions(in)

		verify(fmt, exp, out)
	}
}

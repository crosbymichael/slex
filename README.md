## SLEX

[![Build Status](https://travis-ci.org/crosbymichael/slex.svg?branch=master)](https://travis-ci.org/crosbymichael/slex)

slex is a simple binary that allows you to run a command on multiple hosts via SSH.
It is very similar to fabric except that it is written in Go so you don't have to 
have python installed on your system and you don't *have* to write a script or 
configuration files if you do not want to.

## Building

To build `slex` you must have a working Go install then you can run:

```bash
go get -d github.com/crosbymichael/slex
```

```bash
slex -h
NAME:
   slex - SSH commands multiplexed

USAGE:
   slex [global options] command [command options] [arguments...]

VERSION:
   1

AUTHOR:
  @crosbymichael - <crosbymichael@gmail.com>

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug                     enable debug output for the logs
   --host value                SSH host address
   --hosts value               file containing host addresses separated by a new line
   --user value, -u value      user to execute the command as (default: "root")
   --identity value, -i value  SSH identity to use for connecting to the host
   --option value, -o value    SSH client option
   --agent, -A                 Forward authentication request to the ssh agent
   --env value, -e value       set environment variables for SSH command
   --quiet, -q                 disable output from the ssh command
   --help, -h                  show help
   --version, -v               print the version

```

### Get the uptime for all servers
```bash
slex --host 192.168.1.3 --host 192.168.1.4 uptime
[192.168.1.3:22]  01:05:20 up  4:44,  0 users,  load average: 0.35, 0.39, 0.33
[192.168.1.4:22]  01:05:20 up  9:45,  0 users,  load average: 0.04, 0.07, 0.06
```

### Run a docker container on all servers
```bash
slex --host 192.168.1.3 --host 192.168.1.4 docker run --rm busybox echo "hi slex"
[192.168.1.3:22] hi slex
[192.168.1.4:22] hi slex
```

### Pipe scripts to all servers
```bash
echo "echo hi again" | slex --host 192.168.1.3 --host 192.168.1.4
[192.168.1.3:22] hi again
[192.168.1.4:22] hi again
```

#### License - MIT

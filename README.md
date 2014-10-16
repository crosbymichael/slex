## SLEX

slex is a simple binary that allows you to run a command on multiple hosts via SSH.
It is very similar to fabric except that it is written in Go so you don't have to 
have python installed on your system and you don't *have* to write a script or 
configuration files if you do not want to.


### Get the uptime for all servers
```bash
slex --host 192.168.1.3 --host 192.168.1.4 uptime
[192.168.1.3:22]  01:05:20 up  4:44,  0 users,  load average: 0.35, 0.39, 0.33
[192.168.1.4:22]  01:05:20 up  9:45,  0 users,  load average: 0.04, 0.07, 0.06
```

### Run a docker container on all servers
```bash
slex --host 192.168.1.3 --host 192.168.1.4 docker run --rm busybox echo "hi slex"
[104.131.131.110:22]  hi slex
[198.199.103.188:22]  hi slex
```

#### License - MIT

# Command: enslave

The enslave command provides a more fully capable option for enslaving
one Redis node to another.

Given a pair of nodes it will enslave one to the other. However, it also
can authenticate against each node using different passwords, will set
the masterauth directive in the slave, and can synchronize the passwords
from the master's password as provided by the `-M` or `--masterauth` option.

```
Usage: 
  reditool enslave [flags]

 Available Flags:
  -A, --authsync=false: Wait for sync to complete
      --help=false: help for enslave
  -m, --master="127.0.0.1:6379": Host to slave to
  -M, --masterauth="": The auth token needed to work with the master node
  -s, --slave="127.0.0.1:6379": Host to enslave
  -S, --slaveAuth="": The initial auth token needed to work with the slave node
  -t, --timeout=10: Seconds before a slave sync times out
  -v, --verbose=false: Be verbose in what we log
  -w, --waitsync=false: Wait for sync to complete
```


# Command: enslave


The enslave command provides a more fully capable option for enslaving
one Redis node to another.

Given a pair of nodes it will enslave one to the other. However, it also
can authenticate against each node using different passwords, will set
the masterauth directive in the slave, and can synchronize the passwords
from the master's password as provided by the `-M` or `--masterauth` option.

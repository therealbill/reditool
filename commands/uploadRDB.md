# TODO: Command: uploadRDB


This command will read a local RDB file, fire up a "server", connect to
the specified client, slave the client to this process' IP:Port in order
to synchronize the data. It will then poll the slave to check for a
complete sync. Once sync is complete it will promote the target to
master and exit.

Think of it as a remote restore agent.

Of course, this means the target server will need connectivity to the IP
and Port this command listens on.



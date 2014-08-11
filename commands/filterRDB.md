# TODO: Command: filterRDB

This command will read a local RDB file, fire up a "server", connect to
the specified client, slave the client to this process' IP:Port in order
to synchronize the data. It will then poll the slave to check for a
complete sync. Once sync is complete it will promote the target to
master and exit.

Of course, this means the target server will need connectivity to the IP
and Port this command listens on.

The difference between this and uploadRDB is this command will take an
argument which will be used in a regular expression to filter only
certain keys to the slave. This can be quite useful for migrating or
replicating keys to a specific instance.

I'd also like to make a version/option for this command to connect to a
running Redis instance, slave to it, and proxy the RDB data to the
target.

Additionally, I'd like to have options for filtering based on the
command rather than the key. Not sure why I want to see that other than
it should not be difficult and my gut seems to think it should be
there. Maybe it's to provide more detailed examples of how this
command/function could be useful.

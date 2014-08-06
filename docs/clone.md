# Clone Commands

The clone command provides a simple way of duplicating a Redis node.
Replication can copy data but does not copy setting. Sentinel can be used
to manage slave and to manage a failover, but setting up a sentinel
constellation is a significant amount of effort. Finally, with the
availability of Redis as a Service you may not always have access to the
config files to copy from or to the system to copy a config file in.

For these scenarios, the clone command is quite useful.

To use the clone command you will need a minimum of a target instance
and a slave instance; both of which will need to be running and
accessible from where you run reditool.

# Usage and Options
```
Usage:
  reditool clone [flags]

 Available Flags:
  -o, --origin="127.0.0.1:6379": Host to clone freom to
	The IP of the host you want to clone, known as the origin
  -c, --clone="127.0.0.1:6379": 
	The IP and port of the host you want to clone to
  -p, --promote=false: 
	Promote clone to master when completed.
  -R, --reconfigure=false: 
		Reconfigure slaves to point to the new clone when complete, implies -p
  -r, --role="master": 
		Role the server must present before we perform backup. 
  -t, --timeout=10: 
		Number of seconds to wait before before a slave sync times out
```

## Role Requirements

By default reditool will only perform the clone operation if the origin
is a master as indicated by the info replcation:role display. If you are
cloning a slave, passing `-r slave` will invert this to ensure the clone
is being taken from a slave server.


## Automated Promotion

Using the `-p` flag you can instruct reditool to promote the new clone
to master when configuration sync is complete. This is useful for
migrating a master which is outgrowing it's current host to one with
more resources or for creating a clone of production to be used in
development, testing, or demo purposes.

This option only promotes the the newly cloned server to be a master,
without additional options it does nothing about any slaves currently
attached to the origin. For that, you will need the reconfigure option.

## Slave Reconfiguration

In the event you are migrating a master entirely and have slaves already
attached, you can pass the `-R` option to have reditool automatically
reconfigure the slaves to point to the new master. There are timing
concerns to be aware of when doing this.

If slaves are writing to the master and you clone & promote it you now
have two masters, only one of which is receiving updates. These updates
will be lost when pointing to the new master. You can minimize the risk
of this by using the reconfiguration option to reconfigure the slaves as
soon as the clone has been promoted, but there can be a delay of up to a
second per slave.

Alternatively one could reconfigure the slaves prior to promotion, but
then writes will still be lost as the clone is still syncing from the
master.

Thus, as in all cases, thought should be applied to the process of
migrating a master. You need to analyze your risk potential for the
migration and determine how compfortable you are with the realtively
tiny window of potential write loss.



# RediTool Overview

Reditool is/will be the "swiss army knife for Redis". It will provide
management and inspection utilities for Redis at the commannd line. It
will understand Sentinel and Cluster and be able to do such things as
connect to a Redis instance and upload it's data directly to cloud
storage, or to clone a redis instance.

Reditool is implemented as a master command with subcommands, much like git
works. For example `reditool help <command>` will show help for a given
command.


# Current Features

As new features/commands are added they will be highlighted here.

## Redis Node Cloning

The clone command provides the ability to clone one redis server to another,
including configuration and data. It also provides the ability to promote the
new clone to a master as well as reconfigure origin-attached slaves to point to
the new master.

Note: If the origin instance does not allow or support the CONFIG
command, you cannot clone it. In this case you must enslave the clone to
the master the promote when complete. To do this you pass the `-n` or
`--noconfig` option and clone will skip the config sync portion. Or use
the enslave command.

## Sentinel Node Cloning

The sentinel-clone command provides the ability to duplicate the current
configuration of a Sentinel instance to a new sentinel instance. It can also
handle purging the origin of pods to effect sentinel migration. This will
result in the origin sentinel instance being a blank slate.


# Other Commands

Sometimes commands get added but not mentioned here. You can use
`reditool help` to see a command listing and `reditool help <command>`
to get help on each command.

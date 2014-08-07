# Command: backup


This command makes it easy to connect to  redis server, pull down the
current snapshot and store it somewhere. That somewhere can be a local
file, Rackspace CloudFile or Amazon S3.

# Usage and Options

```
Usage: 
  reditool backup [flags]

 Available Flags:
  -a, --apikey="": The API key for the cloud storage service being used
  -c, --container="redis-backups": The container/directry to store the backup in
  -d, --destination="localfile": Which destination type to save the backup to
      --help=false: help for backup
  -h, --host="127.0.0.1": Host to connect to
  -n, --nameformat="02-01-2006-15-04-dump.rdb": The time format example to use and the suffix. This will result in the name of the file the dump is saved to. For your reference the understood values are 'Mon Jan 2 15:04:05 MST 2006'. To get MM-YYYY-DD.rd use '01-2006-02.rdb'
  -p, --port=6379: Port to connect to
  -r, --role="master": Role the server must present before we perform backup
  -u, --username="": The username for the cloud storage service being used
```


Most of the options should be easy to figure out from the usage text.
However some might seem overloaded.

## Destination: -d | --destination

This is the option which determines if you want to use "localfil" to
store the RDB locally, "cloudfiles" or "s3" for those providers
respectively.

## Name Format: -n | --nameformat

Think of this as a pattern to use when creating the file name. In a
cloud storage provider this will be the name of the object. On the local
filesystem option it will be the name of the file. The format is a
string identifying a timestamp format per the Go standard. Essentially,
pretend the date is 'Mon Jan 2 15:04:05 MST 2006' Now, with that
information construct the string as you would like using those numbers.

For example, to make it a daily backup with YYYY-MM-DD.rd, you would
pass '01-2006-02.rdb' as that is what it would look like on that date.

## Container: -c | --container

For localfile destination this is a directory. For could storage the
container to use.


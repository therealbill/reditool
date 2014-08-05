package commands

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	client "github.com/therealbill/libredis/client"
	destinationDrivers "github.com/therealbill/redis-buagent/drivers"
)

var (
	roleRequired      string
	backupDestination string
	containerName     string
	fileFormat        string
	apikey            string
	username          string
	logger            *log.Logger
)

func init() {
	backup.Flags().IntVarP(&targetPort, "port", "p", 6379, "Port to connect to")
	backup.Flags().StringVarP(&targetHost, "host", "h", "127.0.0.1", "Host to connect to")
	backup.Flags().StringVarP(&roleRequired, "role", "r", "master", "Role the server must present before we perform backup")
	backup.Flags().StringVarP(&backupDestination, "destination", "d", "localfile", "Which destination type to save the backup to")
	backup.Flags().StringVarP(&containerName, "container", "c", "redis-backups", "The container/directry to store the backup in")
	backup.Flags().StringVarP(&fileFormat, "nameformat", "n", "02-01-2006-15-04-dump.rdb", "The time format example to use and the suffix. This will result in the name of the file the dump is saved to. For your reference the understood values are ' Mon Jan 2 15:04:05 MST 2006'. To get MM-YYYY-DD.rd use '01-2006-02.rdb'")
	logger = log.New(os.Stdout, "reditool", log.LstdFlags)
	backup.Flags().StringVarP(&apikey, "apikey", "a", "", "The API key for the cloud storage service being used")
	backup.Flags().StringVarP(&username, "username", "u", "", "The username for the cloud storage service being used")
}

var backup = &cobra.Command{
	Use:   "backup",
	Short: "Backup the redis server remotely",
	Long:  `Connect to the redis server, pull down the current snapshot and store it somewhere.`,
	Run:   BackupServer,
}

type DestinationDriverConfig struct {
	Name              string
	Username          string
	Apikey            string
	Layout            string
	Authurl           string
	Containername     string
	DestinationFormat string
	Logger            *log.Logger
}

func getDriver(driverconfig DestinationDriverConfig) destinationDrivers.Driver {
	switch driverconfig.Name {
	case "cloudfiles":
		mydriver := new(destinationDrivers.CloudFilesDriver)
		mydriver.Name = driverconfig.Name
		mydriver.Username = driverconfig.Username
		mydriver.Apikey = driverconfig.Apikey
		mydriver.Authurl = "https://auth.api.rackspacecloud.com/v1.0"
		mydriver.Layout = driverconfig.DestinationFormat
		mydriver.Containername = driverconfig.Containername
		mydriver.Logger = driverconfig.Logger
		return mydriver

	case "s3":
		mydriver := new(destinationDrivers.AmazonS3Driver)
		mydriver.Name = driverconfig.Name
		mydriver.Username = driverconfig.Username
		mydriver.Apikey = driverconfig.Apikey
		mydriver.Layout = driverconfig.DestinationFormat
		mydriver.Containername = driverconfig.Containername
		mydriver.Logger = driverconfig.Logger
		return mydriver

	case "localfile":
		mydriver := new(destinationDrivers.LocalFileDriver)
		mydriver.Name = driverconfig.Name
		mydriver.Layout = driverconfig.DestinationFormat
		mydriver.Containername = driverconfig.Containername
		mydriver.Logger = driverconfig.Logger
		return mydriver
	}
	return new(destinationDrivers.MissingDriver)
}

func BackupServer(cmd *cobra.Command, args []string) {
	host := cmd.Flags().Lookup("host").Value.String()
	port, _ := strconv.Atoi(cmd.Flags().Lookup("port").Value.String())

	dconfig := DestinationDriverConfig{Name: backupDestination, DestinationFormat: fileFormat, Apikey: apikey, Username: username}
	dconfig.DestinationFormat = fileFormat
	dconfig.Apikey = apikey
	dconfig.Username = username
	dconfig.Logger = logger

	switch backupDestination {
	case "localfile":
		dconfig.Containername = containerName
	case "cloudfiles":
		dconfig.Containername = containerName
		badContainer := strings.Contains(dconfig.Containername, "/")
		badName := strings.Contains(dconfig.DestinationFormat, "/")
		if badName || badContainer {
			log.Fatal("Cloudfiles does not support '/' in container or file names, aborting")
		}
	case "s3":
		dconfig.Containername = containerName
	default:
		logger.Fatal("Unknown backup destination driver given:", backupDestination)
	}

	logger.Println("Backup up to driver:", dconfig.Name)
	r, err := client.Dial(host, port)
	if err != nil {
		logger.Fatal("Unable to connect to redis instance")
	} else {
		logger.Println("Connection to redis confirmed")
	}
	info, err := r.Info()
	role := info.Replication.Role
	if err != nil {
		logger.Fatal("Unable to get the role of the redis instance")
	}
	logger.Println("Role:", role)
	switch role {
	case roleRequired:
		logger.Println("acceptable role confirmed, now to perform a backup...")
	default:
		logger.Println("Role mismatch, no backup will be performed")
		return
	}
	td := getDriver(dconfig)
	//fmt.Println("Backup up to:", td.Containername)
	td.Connect()
	proceed := td.Authenticate()
	if !proceed {
		logger.Fatal("Unable to proceed due to failed Authorize phase")
	}
	res, err := r.ExecuteCommand("SYNC")
	if err != nil {
		logger.Println("Error on sync:", err)
	}
	rdb, err := res.BytesValue()
	if err != nil {
		logger.Println("Error on sync:", err)
	}

	datasize := float64(len(rdb)) / 1024.0
	logger.Printf("Origin data is %.4f Kb\n", float64(datasize))

	td.Upload(rdb)

}

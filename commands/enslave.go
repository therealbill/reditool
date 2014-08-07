package commands

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	client "github.com/therealbill/libredis/client"
)

var (
	masterHost  string
	slaveHost   string
	masterAuth  string
	slaveAuth   string
	waitForSync bool
	syncAuth    bool
)

func init() {
	logger = log.New(os.Stdout, "reditool ", log.LstdFlags)
	enslaveCommand.Flags().StringVarP(&masterHost, "master", "m", "127.0.0.1:6379", "Host to slave to")
	enslaveCommand.Flags().StringVarP(&slaveHost, "slave", "s", "127.0.0.1:6379", "Host to enslave")
	enslaveCommand.Flags().StringVarP(&masterAuth, "masterauth", "M", "", "The auth token needed to work with the master node")
	enslaveCommand.Flags().StringVarP(&slaveAuth, "slaveAuth", "S", "", "The initial auth token needed to work with the slave node")
	enslaveCommand.Flags().BoolVarP(&verbose, "verbose", "v", false, "Be verbose in what we log")
	enslaveCommand.Flags().BoolVarP(&waitForSync, "waitsync", "w", false, "Wait for sync to complete")
	enslaveCommand.Flags().BoolVarP(&syncAuth, "authsync", "A", false, "Wait for sync to complete")
	enslaveCommand.Flags().Float64VarP(&syncTimeout, "timeout", "t", 10, "Seconds before a slave sync times out")
}

var enslaveCommand = &cobra.Command{
	Use:   "enslave",
	Short: "enslave one redis server to another",
	Long:  `COnfigure one Redis server to be slaved to another`,
	Run:   EnslaveServer,
}

// CloneServer does the heavy lifting to enslave one Redis instance to another.
func EnslaveServer(cmd *cobra.Command, args []string) {

	if slaveHost == masterHost {
		log.Fatal("Will not slave a host to itself, aborting")
	}

	// Connect to the Origin node
	masterConf := client.DialConfig{Address: masterHost, Password: masterAuth}
	master, err := client.DialWithConfig(&masterConf)
	if err != nil {
		logger.Fatal("Unable to connect to master")
	} else {
		doLog("Connection to master confirmed")
	}
	// obtain node information
	info, err := master.Info()
	if err != nil {
		logger.Fatal("unable to run commands on master server due to error:", err)
	}
	role := info.Replication.Role
	if len(role) == 0 {
		logger.Fatal("Unable to get the role of the master instance, try authentication")
		return
	}
	// verify the role we get matches our condition for a backup
	switch role {
	case roleRequired:
		doLog("acceptable role confirmed, now to enslave...")
	default:
		doLog("Role mismatch, no enslavement will be performed")
		return
	}
	// Now connect to the slave ...
	slaveConf := client.DialConfig{Address: slaveHost, Password: slaveAuth}
	slave, err := client.DialWithConfig(&slaveConf)
	if err != nil {
		logger.Fatal("Unable to connect to slave")
	} else {
		doLog("Connection to slave confirmed")
	}
	slave.Info()
	// OK, now we are ready to start cloning
	err = slave.ConfigSet("masterauth", masterAuth)
	if err != nil {
		doLog("Unable to set masterauth on slave!")
	}

	if syncAuth {
		err = slave.ConfigSet("requirepass", masterAuth)
		slave.ExecuteCommand("AUTH", masterAuth)
	}

	slaveof := strings.Split(masterHost, ":")
	slaveres := slave.SlaveOf(slaveof[0], slaveof[1])
	if slaveres != nil {
		logger.Printf("Unable to enslave to master! Error: '%s'\n", slaveres)
		doLog("Aborting enslave so you can investigate why.")
		return
	}
	doLog(fmt.Sprintf("Successfully enslaved to %s:%s\n", slaveof[0], slaveof[1]))
	if waitForSync {
		new_info, _ := slave.Info()
		syncComplete := !new_info.Replication.MasterSyncInProgress
		println("SIP:", new_info.Replication.MasterSyncInProgress)
		println("syncComplete:", syncComplete)
		syncTime := 0.0
		if !syncComplete {
			doLog("Sync in progress...")
			for {
				new_info, _ := slave.Info()
				syncComplete = !new_info.Replication.MasterSyncInProgress
				println("SIP:", new_info.Replication.MasterSyncInProgress)
				println("syncComplete:", syncComplete)
				if !syncComplete {
					syncTime += .5
					if syncTime >= syncTimeout {
						doLog("Sync took longer than expected, aborting until this is better handled!")
						break
					}
					time.Sleep(time.Duration(500) * time.Millisecond)
				} else {
					break
				}
			}
		}
		new_info, _ = slave.Info()
		if new_info.Replication.MasterLinkStatus != "up" {
			doLog(fmt.Sprintf("masterlink is not 'up', but '%s'", new_info.Replication.MasterLinkStatus))
			syncComplete = false
		}
		if !syncComplete {
			doLog("Sync did not complete")
			return
		}
		doLog("Sync appears to be completed")
		return
	}
}

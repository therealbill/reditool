package commands

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	client "github.com/therealbill/libredis/client"
)

var (
	originHost          string
	cloneHost           string
	originAuth          string
	cloneAuth           string
	promoteWhenComplete bool
	reconfigureSlaves   bool
	noConfig            bool
	verbose             bool
	syncTimeout         float64
)

func init() {
	logger = log.New(os.Stdout, "reditool ", log.LstdFlags)
	cloneCommand.Flags().StringVarP(&originHost, "origin", "o", "127.0.0.1:6379", "Host to clone freom to")
	cloneCommand.Flags().StringVarP(&cloneHost, "clone", "c", "127.0.0.1:6379", "Host to clone to")
	cloneCommand.Flags().StringVarP(&roleRequired, "role", "r", "master", "Role the server must present before we perform backup")
	cloneCommand.Flags().StringVarP(&originAuth, "originauth", "O", "", "The auth token needed to work with the origin node")
	cloneCommand.Flags().StringVarP(&cloneAuth, "cloneAuth", "C", "", "The initial auth token needed to work with the clone node")
	cloneCommand.Flags().BoolVarP(&promoteWhenComplete, "promote", "p", false, "Promote clone to master when completed")
	cloneCommand.Flags().BoolVarP(&reconfigureSlaves, "reconfigure", "R", false, "Reconfigure slaves to point to the new clone when complete, implies -p")
	cloneCommand.Flags().BoolVarP(&noConfig, "noconfig", "n", false, "This option is used when your origin node does not support or allow the CONFIG command; your origin configuration will not be cloned - just the data.")
	cloneCommand.Flags().BoolVarP(&verbose, "verbose", "v", false, "Be verbose in what we log")
	cloneCommand.Flags().Float64VarP(&syncTimeout, "timeout", "t", 10, "Seconds before a slave sync times out")
}

var cloneCommand = &cobra.Command{
	Use:   "clone",
	Short: "clone one redis server to another",
	Long:  `Given a redis server to clone and another to clone it to, clone the settings and data`,
	Run:   CloneServer,
}

// CloneServer does the heavy lifting to clone one Redis instance to another.
func CloneServer(cmd *cobra.Command, args []string) {

	if cloneHost == originHost {
		log.Fatal("Can not clone a host to itself, aborting")
	}

	// Connect to the Origin node
	originConf := client.DialConfig{Address: originHost, Password: originAuth}
	origin, err := client.DialWithConfig(&originConf)
	if err != nil {
		logger.Fatal("Unable to connect to origin")
	} else {
		doLog("Connection to origin confirmed")
	}
	// obtain node information
	info, err := origin.Info()
	if err != nil {
		logger.Fatal("unable to run commands on origin server due to error:", err)
	}
	role := info.Replication.Role
	if len(role) == 0 {
		logger.Fatal("Unable to get the role of the origin instance, try authentication")
		return
	}
	// verify the role we get matches our condition for a backup
	switch role {
	case roleRequired:
		doLog("acceptable role confirmed, now to perform a clone...")
	default:
		doLog("Role mismatch, no clone will be performed")
		return
	}
	// Now connect to the clone ...
	cloneConf := client.DialConfig{Address: cloneHost, Password: cloneAuth}
	clone, err := client.DialWithConfig(&cloneConf)
	if err != nil {
		logger.Fatal("Unable to connect to clone")
	} else {
		doLog("Connection to clone confirmed")
	}
	clone.Info()
	var oconfig map[string]string
	// OK, now we are ready to start cloning
	err = clone.ConfigSet("masterauth", originAuth)
	if err != nil {
		doLog("Unable to set masterauth on clone!")
	}
	if !noConfig {
		oconfig, _ = CloneConfig(origin, clone)
	}

	switch role {
	case "slave":
		// If we are cloning a slave we are assuming it needs to look just like
		// the others, so we simply clone the settings and slave it to the
		// origin's master
		slaveof := strings.Split(oconfig["slaveof"], " ")
		logger.Printf("Need to set clone to slave to %s on port %s\n", slaveof[0], slaveof[1])
		slaveres := clone.SlaveOf(slaveof[0], slaveof[1])
		if slaveres != nil {
			logger.Printf("Unable to clone slave setting! Error: '%s'\n", slaveres)
		} else {
			logger.Print("Successfully cloned new slave")
			return
		}
	case "master":
		// master clones can get tricky.
		// First, slave to the origin node to get a copy of the data
		doLog("Role being cloned is 'master'")
		doLog("First, we need to slave to the original master to pull data down")
		synced := SyncCloneWithOrigin(originHost, clone, oconfig)
		if !synced {
			doLog("Unable to SYNC with origin, aborting")
			return
		}
		// Now we have synced data.
		// Next we need to see if we should promote the new clone to a master
		// this is useful for migrating a master but also for providing a
		// production clone for dev or testing
		if promoteWhenComplete || reconfigureSlaves {
			promoted := clone.SlaveOf("no", "one")
			if promoted != nil {
				logger.Fatal("Was unable to promote clone to master, investigate why!")
			}
			doLog("Promoted clone to master")
			// IF we are migrating a master entirely, we want to reconfigure
			// it's slaves to point to the new master
			// While it might make sense to promote the clone after slaving,
			// doing that means writes are lost in between slave migration and
			// promotion. This gets tricky, which is why by default we don't do it.
			if !reconfigureSlaves {
				doLog("Not instructed to promote existing slaves")
				logger.Printf("Clone of %s to %s complete", originHost, cloneHost)
				return
			} else {
				logger.Printf("Reconfiguring slaves as requested: %+v", info.Replication.Slaves)
				info, _ := origin.Info()
				slaveof := strings.Split(cloneHost, ":")
				desired_port, _ := strconv.Atoi(slaveof[1])
				for _, data := range info.Replication.Slaves {
					slave_connstring := fmt.Sprintf("%s:%d", data.IP, data.Port)
					logger.Printf("reconfigurign slave '%s", slave_connstring)
					slaveconn, err := client.DialWithConfig(&client.DialConfig{Address: slave_connstring})
					if err != nil {
						logger.Printf("Unable to connect to slave '%s', skipping", slave_connstring)
						continue
					}
					err = slaveconn.SlaveOf(slaveof[0], slaveof[1])
					if err != nil {
						logger.Printf("Unable to slave %s to clone. Err: '%s'", slave_connstring, err)
						continue
					}
					time.Sleep(time.Duration(200) * time.Millisecond) // needed to give the slave time to sync.
					slave_info, _ := slaveconn.Info()
					if slave_info.Replication.MasterHost == slaveof[0] {
						if slave_info.Replication.MasterPort == desired_port {
							logger.Printf(fmt.Sprintf("Slaved %s to clone", slave_connstring))
						} else {
							//doLog(fmt.Sprintf("Hmm, slave settings don't match, look into this on slave %s %d", data.IP, data.Port]))
						}
					}
				}
			}
		}
	}
	logger.Printf("Clone of %s to %s complete", originHost, cloneHost)
}

func CloneConfig(origin, clone *client.Redis) (map[string]string, error) {
	oconfig, err := origin.ConfigGet("*")
	if err != nil {
		logger.Fatal("Unable to get origin config, aborting on err:", err)
		return oconfig, err
	}
	doLog("Cloning config")
	for k, v := range oconfig {
		// slaveof is not clone-able and is set separately, so skip it
		if k == "slaveof" {
			continue
		}
		err := clone.ConfigSet(k, v)
		if err != nil {
			if strings.Contains(err.Error(), "Unsupported CONFIG parameter") {
				if verbose {
					logger.Printf("Setting config parameter '%s' is not supported by redis, not cloned\n", k)
				}
			} else {
				logger.Printf("Unable to set key '%s' to val '%s' on clone due to Error '%s'\n", k, v, err)
			}
		} else {
			if verbose {
				logger.Printf("Key '%s' cloned", k)
			}
		}
		// Now we need to handle the password for the clone changing on account
		// of cloning the requirepass config directive
		if k == "requirepass" {
			res, err := clone.ExecuteCommand("AUTH", v)
			if err != nil || res.Status != "OK" {
				doLog("Unable to authenticate clone after cloning requirepass!")
			}

		}
	}
	doLog("Config cloned, now syncing data")
	return oconfig, nil
}

func SyncCloneWithOrigin(originAddres string, clone *client.Redis, oconfig map[string]string) bool {
	slaveof := strings.Split(originHost, ":")
	slaveres := clone.SlaveOf(slaveof[0], slaveof[1])
	if slaveres != nil {
		logger.Printf("Unable to slave clone to origin! Error: '%s'\n", slaveres)
		doLog("Aborting clone so you can investigate why.")
		return false
	}
	doLog(fmt.Sprintf("Successfully enslaved to %s:%s\n", slaveof[0], slaveof[1]))
	syncComplete := false
	time.Sleep(time.Duration(1000) * time.Millisecond) // wait for the slave to start the sync
	new_info, _ := clone.Info()
	syncComplete = !new_info.Replication.MasterSyncInProgress
	println("SIP:", new_info.Replication.MasterSyncInProgress)
	println("syncComplete:", syncComplete)
	syncTime := 0.0
	if !syncComplete {
		doLog("Sync in progress...")
		for {
			new_info, _ := clone.Info()
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
	new_info, _ = clone.Info()
	if new_info.Replication.MasterLinkStatus != "up" {
		doLog(fmt.Sprintf("masterlink is not 'up', but '%s'", new_info.Replication.MasterLinkStatus))
		syncComplete = false
	}
	if !syncComplete {
		doLog("Sync did not complete")
		return false
	}
	doLog("Sync appears to be completed")
	return true

}

func doLog(msg string) {
	if verbose {
		logger.Print(msg)
	}
}

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
	syncTimeout         float64
)

func init() {
	logger = log.New(os.Stdout, "reditool", log.LstdFlags)
	cloneCommand.Flags().StringVarP(&originHost, "origin", "o", "127.0.0.1:6379", "Host to clone freom to")
	cloneCommand.Flags().StringVarP(&cloneHost, "clone", "c", "127.0.0.1:6379", "Host to clone to")
	cloneCommand.Flags().StringVarP(&roleRequired, "role", "r", "master", "Role the server must present before we perform backup")
	cloneCommand.Flags().StringVarP(&originAuth, "originauth", "O", "", "The auth token needed to work with the origin node")
	cloneCommand.Flags().StringVarP(&cloneAuth, "cloneAuth", "C", "", "The auth token needed to work with the clone node")
	cloneCommand.Flags().BoolVarP(&promoteWhenComplete, "promote", "p", false, "Promote clone to master when completed")
	cloneCommand.Flags().BoolVarP(&reconfigureSlaves, "reconfigure", "R", false, "Reconfigure slaves to point to the new clone when complete, implies -p")
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
		logger.Println("Connection to origin confirmed")
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
	logger.Println("Role:", role)
	// verify the role we get matches our condition for a backup
	switch role {
	case roleRequired:
		logger.Println("acceptable role confirmed, now to perform a clone...")
	default:
		logger.Println("Role mismatch, no clone will be performed")
		return
	}
	// Now connect to the clone ...
	cloneConf := client.DialConfig{Address: cloneHost, Password: cloneAuth}
	clone, err := client.DialWithConfig(&cloneConf)
	if err != nil {
		logger.Fatal("Unable to connect to clone")
	} else {
		logger.Println("Connection to clone confirmed")
	}
	clone.Info()

	oconfig, err := origin.ConfigGet("*")
	if err != nil {
		logger.Fatal("Unable to get origin config, aborting on err:", err)
	}
	// OK, now we are ready to start cloning
	logger.Println("Cloning config")
	for k, v := range oconfig {
		// slaveof is not clone-able and is set separately, so skip it
		if k == "slaveof" {
			continue
		}
		err := clone.ConfigSet(k, v)
		if err != nil {
			if strings.Contains(err.Error(), "Unsupported CONFIG parameter") {
				logger.Printf("Setting config parameter '%s' is not supported by redis, not cloned\n", k)
			} else {
				logger.Printf("Unable to set key '%s' to val '%s' on clone due to Error '%s'\n", k, v, err)
			}
		} else {
			logger.Printf("Key '%s' cloned", k)
		}
	}
	logger.Println("Config cloned, now syncing data")
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
			logger.Println("Successfully cloned new slave")
			return
		}
	case "master":
		// master clones can get tricky.
		// First, slave to the origin nde to get a copy of the data
		logger.Println("Role being cloned is 'master'")
		logger.Println("First, we need to slave to the original master to pull data down")
		slaveof := strings.Split(originHost, ":")
		slaveres := clone.SlaveOf(slaveof[0], slaveof[1])
		if slaveres != nil {
			logger.Printf("Unable to slave clone to origin! Error: '%s'\n", slaveres)
			logger.Println("Aborting clone so you can investigate why.")
			return
		}
		logger.Printf("Successfully cloned to %s:%s\n", slaveof[0], slaveof[1])
		syncComplete := false
		new_info, _ := clone.Info()
		syncComplete = !new_info.Replication.MasterSyncInProgress
		syncTime := 0.0
		if !syncComplete {
			logger.Println("Sync in progress...")
			for {
				new_info, _ := clone.Info()
				syncComplete = !new_info.Replication.MasterSyncInProgress
				if !syncComplete {
					syncTime += .5
					if syncTime >= syncTimeout {
						break
					}
					time.Sleep(time.Duration(500) * time.Millisecond)
				} else {
					break
				}
			}
		}
		if !syncComplete {
			logger.Println("Sync took longer than expected, aborting until this is better handled!")
			return
		}
		logger.Println("Sync appears to be completed")
		// Now we have synced data.
		// Next we need to see if we should promote the new clone to a master
		// this is useful for migrating a master but also for providing a
		// production clone for dev or testing
		if promoteWhenComplete {
			promoted := clone.SlaveOf("no", "one")
			if promoted != nil {
				logger.Fatal("Was unable to promote clone to master, investigate why!")
			}
			logger.Println("Promoted clone to master")
			// IF we are migrating a master entirely, we want to reconfigure
			// it's slaves to point to the new master
			// While it might make sense to promote the clone after slaving,
			// doing that means writes are lost in between slave migration and
			// promotion. This gets tricky, which is why by default we don't do it.
			if !reconfigureSlaves {
				logger.Println("Not instructed to promote existing slaves")
				logger.Println("Clone complete")
				return
			} else {
				// I don't like how this looks but it works
				info, _ := origin.Info()
				slaveof := strings.Split(cloneHost, ":")
				desired_port, _ := strconv.Atoi(slaveof[1])
				for index, data := range info.Replication.Slaves {
					logger.Printf("Reconfiguring slave %d/%d\n", index, info.Replication.ConnectedSlaves)
					slaveMap := make(map[string]string)
					for _, line := range strings.Split(data, ",") {
						dsplit := strings.Split(line, "=")
						slaveMap[dsplit[0]] = dsplit[1]
					}
					slave_connstring := fmt.Sprintf("%s:%s", slaveMap["ip"], slaveMap["port"])
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
					time.Sleep(time.Duration(100) * time.Millisecond) // needed to give the slave time to sync.
					slave_info, _ := slaveconn.Info()
					if slave_info.Replication.MasterHost == slaveof[0] {
						if slave_info.Replication.MasterPort == desired_port {
							logger.Printf("Slaved %s to clone", slave_connstring)
						} else {
							logger.Println("Hmm, slave settings don't match, look into this on slave", slaveMap["ip"], slaveMap["port"])
						}
					}
				}
			}
		}
	}
}

package commands

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	client "github.com/therealbill/libredis/client"
)

var (
	ignoreTilt  bool
	purgeOrigin bool
)

func init() {
	logger = log.New(os.Stdout, "reditool", log.LstdFlags)
	sentinelCloneCommand.Flags().StringVarP(&originHost, "origin", "o", "127.0.0.1:6379", "Host to clone freom to")
	sentinelCloneCommand.Flags().StringVarP(&cloneHost, "clone", "c", "127.0.0.1:6379", "Host to clone to")
	sentinelCloneCommand.Flags().StringVarP(&roleRequired, "role", "r", "master", "Role the server must present before we perform backup")
	sentinelCloneCommand.Flags().BoolVarP(&purgeOrigin, "purge", "p", false, "Purge origin when completed")
	sentinelCloneCommand.Flags().BoolVarP(&ignoreTilt, "ignore-tilt", "i", false, "Ignore tilt mode. By default we will not clone a sentinel in tilt mode.")
	sentinelCloneCommand.Flags().Float64VarP(&syncTimeout, "timeout", "t", 10, "Seconds before a slave sync times out")
}

var sentinelCloneCommand = &cobra.Command{
	Use:   "sentinel-clone",
	Short: "clone one sentinel server to another",
	Long:  `Given a sentinel server to clone and another to clone it to, clone the current configuration`,
	Run:   CloneSentinel,
}

// CloneSentinel copies one sentinel node to another
func CloneSentinel(cmd *cobra.Command, args []string) {

	if cloneHost == originHost {
		log.Fatal("Can not clone a host to itself, aborting")
	}

	// Connect to the Origin node
	originConf := client.DialConfig{Address: originHost}
	origin, err := client.DialWithConfig(&originConf)
	if err != nil {
		logger.Fatal("Unable to connect to origin")
	} else {
		logger.Println("Connection to origin confirmed")
	}
	// obtain node information
	origin_info, err := origin.SentinelInfo()
	if err != nil {
		logger.Fatal("Unable to get redis_mode role of the origin instance")
		return
	}
	mode := origin_info.Server.Mode

	if mode != "sentinel" {
		logger.Printf("info: %+v\n", origin_info.Server)
		logger.Fatal("Origin host is not a Sentinel instance, aborting.")
		return
	}

	// Now connect to the clone ...
	cloneConf := client.DialConfig{Address: cloneHost}
	clone, err := client.DialWithConfig(&cloneConf)
	if err != nil {
		logger.Fatal("Unable to connect to clone")
	} else {
		logger.Println("Connection to clone confirmed")
	}
	clone_info, err := clone.SentinelInfo()
	clone_mode := clone_info.Server.Mode
	if err != nil {
		logger.Fatal("Unable to get redis_mode role of the clone instance")
		return
	}

	if clone_mode != "sentinel" {
		logger.Fatal("Clone host is not a Sentinel instance, aborting.")
		return
	}

	// OK, now we are ready to start cloning
	numFailedTransfers := 0
	sentinels, _ := origin.SentinelMasters()
	for _, pod := range sentinels {
		err := clone.SentinelMonitor(pod.Name, pod.IP, pod.Port, pod.Quorum)
		if err != nil {
			logger.Printf("Unable to add pod '%s' to clone.; Error:'%s'\n", pod.Name, err)
			numFailedTransfers++
		} else {
			logger.Println("Transfered pod:", pod.Name)
		}

	}
	logger.Printf("Pod copy complete, there were %d failed copies", numFailedTransfers)

	if purgeOrigin {
		if numFailedTransfers > 0 {
			logger.Println("Purge attempt will be aborted as there are failed pod transfers.")
		} else {
			for _, pod := range sentinels {
				err := origin.SentinelRemove(pod.Name)
				if err != nil {
					logger.Printf("Unable to purge '%s'; error:'%s'", pod.Name, err)
				} else {
					logger.Printf("Purged '%s'", pod.Name)
				}
			}
		}
	}

}

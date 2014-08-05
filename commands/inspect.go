package commands

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"text/template"

	"github.com/spf13/cobra"
	client "github.com/therealbill/libredis/client"
	rinfo "github.com/therealbill/libredis/info"
)

var targetHost string
var targetPort int

func init() {
	inspectCommand.Flags().IntVarP(&targetPort, "port", "p", 6379, "Port to connect to")
	inspectCommand.Flags().StringVarP(&targetHost, "host", "h", "127.0.0.1", "Host to connect to")
}

var inspectCommand = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect a Redis Server",
	Long:  `This command will inspect a Redis server to various levels of detail and display the resulting information.`,
	Run:   InspectServer,
}

const plainall = `
==[Server]==
Version: {{.Server.Version}}
Port: {{.Server.TCPPort}}
Mode: {{.Server.Mode}}
Uptime (days): {{.Server.UptimeInDays}} 
Uptime (secs):  {{.Server.UptimeInSeconds}}

Resource Consumption:
Memory Used: {{.Memory.UsedMemoryHuman}}

Replication Information
Role: {{.Replication.Role}}
Number of Connected Slaves: {{.Replication.ConnectedSlaves}}

Persistence Configuration
Loading Data From Disk: {{.Persistence.Loading}}
Unsaved Changes: {{.Persistence.ChangesSinceSave}}
{{ if not .Persistence.AOFEnabled }}
Append Only is disabled
{{ end }}

`

func showRawPlain(info rinfo.RedisInfoAll) {

	//fmt.Printf("Server:\n%+v\n", info.Server)
	//fmt.Printf("Memory:\n%+v\n", info.Memory)
	//fmt.Printf("Replication:\n%+v\n", info.Replication)
	fmt.Printf("Persistence:\n%+v\n", info.Persistence)

	t := template.Must(template.New("all").Parse(plainall))
	err := t.Execute(os.Stdout, info)
	if err != nil {
		println()
		log.Fatal("ERROR: Template parsing error:", err)
		println()
	}
	println()
}

func InspectServer(cmd *cobra.Command, args []string) {
	host := cmd.Flags().Lookup("host").Value.String()
	port, _ := strconv.Atoi(cmd.Flags().Lookup("port").Value.String())
	println("Inspecting server", host, port)

	connection, err := client.Dial(host, port)
	if err != nil {
		log.Fatal("Unable to connect to Redis node")
	}
	info, err := connection.Info()
	if err != nil {
		log.Fatal("Unable to call info in Redis, Need to auth first?")
	}
	showRawPlain(info)
}

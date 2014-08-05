package main

import (
	"runtime"

	"github.com/therealbill/reditool/commands"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	commands.Execute()
}

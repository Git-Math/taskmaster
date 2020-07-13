package tasks

import (
	"fmt"
	"os/exec"
	"taskmaster/parse_yaml"
)

var Daemons []exec.Cmd

func Execute(program_map map[string]parse_yaml.Program) {
	fmt.Println("I execute the processes and store Cmds in Daemons")
}

package tasks

import (
	"fmt"
	"os/exec"
	"taskmaster/parse_yaml"
)

type Daemon struct  {
	Name string
	Command exec.Cmd
}

var Daemons []Daemon

func Execute(program_map parse_yaml.ProgramMap) {
	fmt.Println("I execute the processes and store Cmds in Daemons")
}

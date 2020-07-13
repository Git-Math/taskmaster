package master

import (
	"fmt"
	"taskmaster/parse_yaml"
	"taskmaster/tasks"
	"time"
)

func Watch(program_map map[string]parse_yaml.Program) {
	for date := range time.Tick(2000 * time.Millisecond) {
		fmt.Println("I watch the processes. ", date)
		for _, dae := range tasks.Daemons {
			fmt.Println("I watch the process `", dae.Path, "'")
		}
	}
}

package master

import (
	"fmt"
	"taskmaster/tasks"
	"time"
)

func Watch() {
	for date := range time.Tick(2000 * time.Millisecond) {
		fmt.Println("I watch the processes. ", date)
		for _, dae := range tasks.Daemons {
			fmt.Println("I watch the process `", dae.Path, "'")
		}
	}
}

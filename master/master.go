package master

import (
	"fmt"
	"taskmaster/parse_yaml"
	"taskmaster/tasks"
	"time"
)

func Watch(programs_cfg parse_yaml.ProgramMap) {
	for date := range time.Tick(2000 * time.Millisecond) {
		for _, daemon := range tasks.Daemons {
			select {
			case e := <-daemon.Err:
				if e != nil {
					tasks.Register(daemon, "Exited ("+e.Error()+")")
				} else {
					tasks.Register(daemon, "Exited ("+"Success"+")")
				}
				daemon.Stop()
			default:
			}

			cfg := programs_cfg[daemon.Name]
			switch cfg.Autorestart {
			default:
				fmt.Println(date)
			}
		}
	}
}

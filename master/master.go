package master

import (
	"taskmaster/parse_yaml"
	"taskmaster/tasks"
	"time"
)

func Watch(programs_cfg parse_yaml.ProgramMap) {
	for range time.Tick(2000 * time.Millisecond) {
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

			if !daemon.Running {
				cfg := programs_cfg[daemon.Name]
				switch cfg.Autorestart {
				case "unexpected":
					restart := true
					for exitSuccess := range cfg.Exitcodes {
						if exitSuccess == daemon.ExitCode {
							restart = false
							break
						}
					}
					if restart {
						tasks.StartProgram(cfg, daemon)
					}
				case "always":
					tasks.StartProgram(cfg, daemon)
				case "never":
				}

			}
		}
	}
}

package master

import (
	"strconv"
	"taskmaster/parse_yaml"
	"taskmaster/tasks"
	"time"
)

func Watch(programs_cfg parse_yaml.ProgramMap) {
	for range time.Tick(1000 * time.Millisecond) {
		for _, daemons := range tasks.Daemons {
			for _, daemon := range daemons {
				if daemon.Dead {
					continue
				}

				cfg := programs_cfg[daemon.Name]
				uptime := (tasks.CurrentTimeMillisecond() - daemon.StartTime) / 1000
				exited := false
				var err error = nil

				select {
				case err = <-daemon.Err:
					exited = true
				default:
				}

				/* daemon exited before StartTime, check if it needs restarting */
				if uptime < int64(cfg.Starttime) && exited {
					if daemon.StartRetries == cfg.Startretries {
						msg := "Failed to start after " + strconv.Itoa(cfg.Startretries) + " times"
						if err != nil {
							msg += ": " + err.Error()
						}
						tasks.Register(daemon, msg)
						daemon.Dead = true
					}
					if daemon.StartRetries < cfg.Startretries {
						go daemon.Start(cfg)
					}
					continue
				}

				/* daemon just passed StartTime */
				if uptime >= int64(cfg.Starttime) && daemon.Running && !exited {
					tasks.Register(daemon, "Started")
					daemon.Running = true
				}

				if exited {
					if err != nil {
						tasks.Register(daemon, "Exited ("+err.Error()+")")
					} else {
						tasks.Register(daemon, "Exited ("+"Success"+")")
						daemon.ExitCode = 0
					}

					restart := false
					switch cfg.Autorestart {
					case "unexpected":
						success := false
						for exitSuccess := range cfg.Exitcodes {
							if exitSuccess == daemon.ExitCode {
								success = true
								break
							}
						}
						if !success {
							restart = true
						}
					case "always":
						restart = true
					case "never":
					}

					if restart {
						go daemon.Start(cfg)
					} else {
						daemon.Dead = true
					}
				}
			}

		}
	}
}

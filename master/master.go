package master

import (
	"fmt"
	"log"
	"os"
	"taskmaster/parse_yaml"
	"taskmaster/tasks"
	"time"
)

func Watch(programs_cfg parse_yaml.ProgramMap) {
	for date := range time.Tick(2000 * time.Millisecond) {
		registerFile, err := os.Create("register.log")
		if err != nil {
			log.Fatal(err)
		}
		defer registerFile.Close()

		fmt.Println("I watch the processes. ", date)
		for _, dae := range tasks.Daemons {
			fmt.Println("I watch the process `", dae.Name, "'")
			cfg := programs_cfg[dae.Name]

			_, err := os.FindProcess(dae.Command.Process.Pid)
			stopped := err != nil
			if stopped && !dae.Stopped {
				exitCode := dae.Command.Process.ExitCode()
				exitSuccess := false
				for _, successfullExitCode := range cfg.ExitCodes {
					if successfullExitCode == exitCode {
						exitSuccess = true
						break
					}
				}

				/* log to register */
				status := "Stopped, Exit Code: " + dae.Command.Process.ExitCode()
				if exitSuccess {
					status += " (Success)"
				} else {
					status += " (Failure)"
				}
				fmt.Fprintln(registerFile, date, ":", status)

				/* restart if needed */
				switch cfg.Autorestart {
				case "unexpected":
					if exitSuccess {
						break
					}
				case "always":
					err := dae.Start()
					if err != nil {
						log.Fatal(err)
					}
					/* log to register */
					fmt.Fprintln(registerFile, date, ":", "Started")
					break
				case "never":
					break
				}
			}
		}
	}
}

package parse_yaml

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
)

type Program struct {
	Cmd          string            `yaml:"cmd"`
	Numprocs     int               `yaml:"numprocs"`
	Umask        string            `yaml:"umask"`
	Workingdir   string            `yaml:"workingdir"`
	Autostart    string            `yaml:"autostart"`
	Autorestart  string            `yaml:"autorestart"`
	Exitcodes    []int             `yaml:"exitcodes"`
	Startretries int               `yaml:"startretries"`
	Starttime    int               `yaml:"starttime"`
	Stopsignal   string            `yaml:"stopsignal"`
	Stoptime     int               `yaml:"stoptime"`
	Stdout       string            `yaml:"stdout"`
	Stderr       string            `yaml:"stderr"`
	Env          map[string]string `yaml:"env"`
}

type ProgramMap = map[string]Program

type ProgramsStruct struct {
	Programs ProgramMap `yaml:"programs"`
}

func parse_yaml(yamlfile string) ProgramMap {
	var programs_struct ProgramsStruct

	yaml_data, err := ioutil.ReadFile(yamlfile)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(yaml_data, &programs_struct)
	if err != nil {
		log.Fatal(err)
	}

	return programs_struct.Programs
}

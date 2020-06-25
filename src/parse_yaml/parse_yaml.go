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

type ProgramsStruct struct {
	Programs map[string]Program `yaml:"programs"`
}

func parse_yaml(filename string) map[string]Program {
	var programs_struct ProgramsStruct

	yaml_data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(yaml_data, &programs_struct)
	if err != nil {
		log.Fatal(err)
	}

	return programs_struct.Programs
}

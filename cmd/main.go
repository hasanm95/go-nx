package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"runtime"

	"github.com/hasanm95/go-nx/internal/config"
)

type FlagVars struct {
	ConfigPath string
}

func SetupFlags() FlagVars{
	var fg FlagVars
	flag.StringVar(&fg.ConfigPath, "config", "config.yml", "path to the config YAML file")	
	
	flag.Parse()
	return fg;
}

func main() {
	fg := SetupFlags()
	
	cfg, err := config.ParseConfig(fg.ConfigPath)

	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	var workers int

	if cfg.Server.Workers == 0 {
		workers = runtime.NumCPU()
	} else {
		workers = cfg.Server.Workers
	}

	log.Printf("resolved worker count: %d", workers)

	role := os.Getenv("GONX_ROLE")


	if role == "" {
		log.Print("Running as master")
		execPath, err := os.Executable()

		if err != nil {
			log.Fatal("Error get current executatble")
		}
		
		var cmds []*exec.Cmd

		for i := 0; i < workers; i++ {
			cmd := exec.Command(execPath, "--config", fg.ConfigPath)
			cmd.Env = append(os.Environ(), "GONX_ROLE=worker")
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Start()
			if err != nil {
				log.Printf("failed to start worker %d: %v", i, err)
				continue
			}
    		cmds = append(cmds, cmd)
		}

		for _, cmd := range cmds {
			if err := cmd.Wait(); err != nil {
				log.Printf("worker exited with error: %v", err)
			}
		}
	} else {
		log.Print("Running as worker")
	}

	log.Printf("%+v", cfg)
}
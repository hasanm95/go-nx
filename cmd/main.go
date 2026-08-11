package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/hasanm95/go-nx/internal/config"
	"golang.org/x/sys/unix"
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
		fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, 0)
		if err != nil {
			log.Fatalf("failed to crate socket: %v", err)
		}
		defer unix.Close(fd) 
		err = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		if err != nil {
			log.Fatalf("Failed to apply socket configuration: %v", err)
		}
		
		addr := &unix.SockaddrInet4{Port: cfg.Server.Listen}
		copy(addr.Addr[:], []byte{127, 0, 0, 1})

		err = unix.Bind(fd, addr)
		if err != nil {
			log.Fatalf("Failed to bind socket: %v", err)
		}

		err = unix.Listen(fd, 128)
		if err != nil {
			log.Fatalf("Failed to listen: %v", err)
		}

		fmt.Println("Raw socket listening on 127.0.0.1:8080...")

		f := os.NewFile(uintptr(fd), "gonx-listener")
		ln, err := net.FileListener(f)

		if err != nil {
			log.Fatalf("Failed to listen fd: %v", err)
		}

		f.Close()

		handlerFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Fetch the current process ID dynamically on every request
			pid := os.Getpid()
			
			// Write the plain text string response back to the client
			responseString := fmt.Sprintf("hello from worker, pid %d\n", pid)
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(responseString))
		})

		log.Printf("[PID: %d] Starting HTTP server on reused port...", os.Getpid())

		err = http.Serve(ln, handlerFunc)
		if err != nil {
			log.Fatalf("Server stopped unexpectedly: %v", err)
		}
	}

	log.Printf("%+v", cfg)
}
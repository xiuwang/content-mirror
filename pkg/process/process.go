package process

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/openshift/content-mirror/pkg/reaper"
)

type Process struct {
	path string

	configAvailable chan struct{}
}

// New starts and manages a nginx child process that should be reloaded when
// notified. It uses the provided config path and will not load until the first time Reload()
// is called.
func New(configPath string) *Process {
	return &Process{
		path:            configPath,
		configAvailable: make(chan struct{}, 1),
	}
}

func (w *Process) Reload() {
	select {
	case w.configAvailable <- struct{}{}:
	default:
	}
}

func (w *Process) Run() {
	go func() {
		reaper.Start()

		// wait for the first configuration to be available
		<-w.configAvailable
		if out, err := exec.Command("nginx", "-c", w.path, "-t").CombinedOutput(); err != nil {
			if _, ok := err.(*exec.ExitError); ok {
				fmt.Fprintf(os.Stderr, "error: the generated configuration is not valid:\n%s", string(out))
			} else {
				fmt.Fprintf(os.Stderr, "error: unable to execute command: %v", err)
			}
			os.Exit(1)
		}

		log.Printf("Starting proxy ...")
		exits := 0
		for {
			if err := w.runOnce(); err != nil {
				log.Printf("error: %v", err)
				time.Sleep(time.Second)
			} else {
				log.Printf("warn: process exited without error")
			}
			exits++
			if exits > 5 {
				log.Printf("error: Proxy process has exited too many times, crashing")
				os.Exit(1)
			}
		}
	}()
}

func (w *Process) runOnce() error {
	cmd := exec.Command("nginx", "-c", w.path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error)
	go func() {
		defer close(done)
		done <- cmd.Wait()
	}()

	for {
		select {
		case err := <-done:
			return err
		case _, ok := <-w.configAvailable:
			if !ok {
				return nil
			}
			if err := cmd.Process.Signal(syscall.SIGHUP); err != nil {
				log.Printf("error: unable to signal command: %v", err)
			}
		}
	}
}

package project

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/logger"
	"github.com/sgq995/nova/internal/module"
)

type runner struct {
	config *config.Config

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	wg sync.WaitGroup
}

func newRunner(c *config.Config) *runner {
	return &runner{config: c}
}

func (r *runner) start(files map[string][]byte) error {
	main := filepath.Join(module.Root(), r.config.Codegen.OutDir, "main.go")
	cmd := exec.Command("go", "run", main)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	r.cmd = cmd
	r.stdin = stdin
	r.stdout = stdout
	r.stderr = stderr

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		buf := make([]byte, 1024)
		for {
			n, err := r.stdout.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				logger.Debugf("%s", string(buf[:n]))
			}

			if r.cmd.ProcessState != nil {
				return
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		buf := make([]byte, 1024)
		for {
			n, err := r.stderr.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				logger.Errorf("%s", string(buf[:n]))
			}

			if r.cmd.ProcessState != nil {
				return
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()

	err = r.cmd.Start()
	if err != nil {
		return err
	}

	for name, contents := range files {
		r.update(name, contents)
	}

	logger.Infof("server listening at http://%s:%d", r.config.Server.Host, r.config.Server.Port)

	return nil
}

func (r *runner) stop() {
	r.cmd.Process.Signal(os.Interrupt)
	syscall.Kill(-r.cmd.Process.Pid, syscall.SIGINT)
	r.cmd.Wait()
	r.wg.Wait()
}

func (r *runner) restart(files map[string][]byte) error {
	r.stop()
	return r.start(files)
}

func (r *runner) update(name string, contents []byte) error {
	logger.Debugf("[runner] update %s", name)
	writer := bufio.NewWriter(r.stdin)
	command := fmt.Sprintf("UPDATE %s %d\n", name, len(contents))
	_, err := writer.WriteString(command)
	if err != nil {
		return err
	}
	_, err = writer.Write(contents)
	if err != nil {
		return err
	}
	return writer.Flush()
}

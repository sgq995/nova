package project

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/module"
)

type runner struct {
	config *config.Config

	cmd   *exec.Cmd
	stdin io.WriteCloser
}

func newRunner(c *config.Config) *runner {
	return &runner{config: c}
}

func (r *runner) create() error {
	main := filepath.Join(module.Root(), r.config.Codegen.OutDir, "main.go")
	cmd := exec.Command("go", "run", main)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	r.cmd = cmd
	r.stdin = stdin

	return nil
}

func (r *runner) start() error {
	err := r.cmd.Start()
	if err != nil {
		return err
	}
	return nil
}

func (r *runner) stop() {
	r.cmd.Process.Signal(os.Interrupt)
	syscall.Kill(-r.cmd.SysProcAttr.Pgid, syscall.SIGINT)
	r.cmd.Wait()
}

func (r *runner) update(filename, contents string) error {
	writer := bufio.NewWriter(r.stdin)
	command := fmt.Sprintf("UPDATE %s %d\n%s", filename, len(contents), contents)
	_, err := writer.WriteString(command)
	if err != nil {
		return err
	}
	return writer.Flush()
}

package project

import (
	"bufio"
	"fmt"
	"io"
	"log"
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

func (r *runner) start(files map[string][]byte) error {
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

	err = r.cmd.Start()
	if err != nil {
		return err
	}

	for name, contents := range files {
		r.update(name, contents)
	}

	return nil
}

func (r *runner) stop() {
	r.stdin.Close()

	r.cmd.Process.Signal(os.Interrupt)
	syscall.Kill(-r.cmd.Process.Pid, syscall.SIGINT)
	r.cmd.Wait()
}

func (r *runner) restart(files map[string][]byte) {
	r.stop()
	r.start(files)
}

func (r *runner) update(name string, contents []byte) error {
	log.Println("[runner]", name)
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

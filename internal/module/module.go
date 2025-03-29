package module

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/sgq995/nova/internal/fsys"
	"github.com/sgq995/nova/internal/must"
)

var root string = must.Must(projectRoot())
var modName string = must.Must(moduleName())

func projectRoot() (string, error) {
	root := must.Must(os.Getwd())

	var goMod string
	for root != filepath.Dir(root) {
		goMod = filepath.Join(root, "go.mod")
		exists := must.Must(fsys.FileExists(goMod))
		if exists {
			return root, nil
		}

		root = filepath.Dir(root)
	}

	goMod = filepath.Join(root, "go.mod")
	exists := must.Must(fsys.FileExists(goMod))
	if exists {
		return root, nil
	}

	return "", errors.New("nova: golang project not found")
}

func moduleName() (string, error) {
	goMod := filepath.Join(root, "go.mod")

	file := must.Must(os.Open(goMod))
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				return parts[1], nil
			}
		}
	}

	return "", errors.New("nova: module name not found")
}

func Root() string {
	return root
}

func ModuleName() string {
	return modName
}

func Abs(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return Join(path)
}

func Rel(targpath string) string {
	return must.Must(filepath.Rel(root, targpath))
}

func Join(elem ...string) string {
	sub := filepath.Join(elem...)
	return filepath.Join(root, sub)
}

package module

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/sgq995/nova/internal/utils"
)

var root string = utils.Must(projectRoot())
var modName string = utils.Must(moduleName())

func projectRoot() (string, error) {
	root := utils.Must(os.Getwd())

	var goMod string
	for root != filepath.Dir(root) {
		goMod = filepath.Join(root, "go.mod")
		exists := utils.Must(utils.FileExists(goMod))
		if exists {
			return root, nil
		}

		root = filepath.Dir(root)
	}

	goMod = filepath.Join(root, "go.mod")
	exists := utils.Must(utils.FileExists(goMod))
	if exists {
		return root, nil
	}

	return "", errors.New("nova: golang project not found")
}

func moduleName() (string, error) {
	goMod := filepath.Join(root, "go.mod")

	file := utils.Must(os.Open(goMod))
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

func Join(elem ...string) string {
	sub := filepath.Join(elem...)
	return filepath.Join(root, sub)
}

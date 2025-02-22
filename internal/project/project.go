package project

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var root string = must(projectRoot())
var modName string = must(moduleName())

func must[T any](obj T, err error) T {
	if err != nil {
		panic(err)
	}
	return obj
}

func fileExists(filename string) (bool, error) {
	_, err := os.Stat(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func projectRoot() (string, error) {
	root := must(os.Getwd())

	var goMod string
	for root != filepath.Dir(root) {
		goMod = filepath.Join(root, "go.mod")
		exists := must(fileExists(goMod))
		if exists {
			return root, nil
		}

		root = filepath.Dir(root)
	}

	goMod = filepath.Join(root, "go.mod")
	exists := must(fileExists(goMod))
	if exists {
		return root, nil
	}

	return "", errors.New("nova: golang project not found")
}

func moduleName() (string, error) {
	goMod := filepath.Join(root, "go.mod")

	file := must(os.Open(goMod))
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

func Abs(dir string) string {
	target := dir
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, dir)
	}
	return target
}

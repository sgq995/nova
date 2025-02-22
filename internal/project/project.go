package project

import (
	"errors"
	"os"
	"path/filepath"
)

var root string = must(projectRoot())

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
	root, err := os.Getwd()
	if err != nil {
		return "", err
	}

	var goMod string
	for len(root) > 1 {
		goMod = filepath.Join(root, "go.mod")
		exists, err := fileExists(goMod)
		if err != nil {
			return "", err
		}

		if exists {
			return root, nil
		}

		root = filepath.Dir(root)
	}

	return "", nil
}

func Root() string {
	return root
}

func Abs(dir string) string {
	target := dir
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, dir)
	}
	return target
}

package esbuild

import (
	"errors"
	"fmt"

	"github.com/evanw/esbuild/pkg/api"
)

func esbuildError(messages []api.Message) error {
	errs := []error{}
	for _, msg := range messages {
		text := fmt.Sprintf("%s:%d: %s", msg.Location.File, msg.Location.Line, msg.Text)
		errs = append(errs, errors.New(text))
	}
	return errors.Join(errs...)
}

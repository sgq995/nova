package codegen

import (
	"github.com/sgq995/nova/internal/config"
)

type Codegen struct {
	config *config.Config
}

func NewCodegen(c *config.Config) *Codegen {
	return &Codegen{config: c}
}

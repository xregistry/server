package xrlib

import (
	"fmt"

	// log "github.com/duglin/dlog"
	. "github.com/xregistry/server/common"
)

type CLIBackend struct {
	XRConfig *Config
}

var _ Backend = &CLIBackend{}

func NewCLIBackend(c *Config) Backend {
	return &CLIBackend{
		XRConfig: c,
	}
}

func (CLIBE *CLIBackend) NewTx(tx *Tx) *XRError {
	return nil
}

func (CLIBE *CLIBackend) FindRegistry(tx *Tx, config *Config, id string,
	accessMode int) (*Registry, *XRError) {

	panic("not yet")
	return nil, nil
}

func (CLIBE *CLIBackend) RegisterEntity(e *Entity) *XRError {
	// Nothing to do for in-memory
	return nil
}

func (CLIBE *CLIBackend) RefreshEntity(e *Entity, accessMode int) *XRError {
	// Nothing to do for in-memory
	return nil
}

func (CLIBE *CLIBackend) GetResourceContents(e *Entity) []byte {
	if e.Type == ENTITY_RESOURCE || e.Type == ENTITY_VERSION {
		return []byte("Placeholder")
	}
	panic(fmt.Sprintf("%s: I'm not a Resource or Version", e.XID))
}

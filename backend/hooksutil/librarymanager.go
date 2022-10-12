package hooksutil

import (
	"io"
	"plugin"

	"github.com/pkg/errors"
	"isc.org/stork/hooks"
)

type LibraryManager struct {
	p *plugin.Plugin
}

func NewLibraryManager(path string) (*LibraryManager, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot open a plugin: %s", path)
	}

	return &LibraryManager{p}, nil
}

func (lm *LibraryManager) Load() (io.Closer, error) {
	symbolName := hooks.HookLoadFunctionName
	symbol, err := lm.p.Lookup(symbolName)
	if err != nil {
		return nil, errors.Wrapf(err, "lookup for symbol: %s failed", symbolName)
	}
	load, ok := symbol.(hooks.HookLoadFunction)
	if !ok {
		return nil, errors.Errorf("symbol %s has unexpected signature", symbolName)
	}

	callouts, err := load()
	err = errors.Wrap(err, "cannot load the hook")

	return callouts, err
}

func (lm *LibraryManager) Version() (program string, version string, err error) {
	symbolName := hooks.HookVersionFunctionName
	symbol, err := lm.p.Lookup(symbolName)
	if err != nil {
		err = errors.Wrapf(err, "lookup for symbol: %s failed", symbolName)
		return
	}
	versionFunction, ok := symbol.(hooks.HookVersionFunction)
	if !ok {
		err = errors.Errorf("symbol %s has unexpected signature", symbolName)
		return
	}
	program, version = versionFunction()
	return
}

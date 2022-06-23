package hooks

import (
	"plugin"

	"github.com/pkg/errors"
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

func (lm *LibraryManager) Load() (interface{}, error) {
	symbolName := HookLoadFunctionName
	symbol, err := lm.p.Lookup(symbolName)
	if err != nil {
		return nil, errors.Wrapf(err, "lookup for symbol: %s failed", symbolName)
	}
	load, ok := symbol.(HookLoadFunction)
	if !ok {
		return nil, errors.Errorf("symbol %s has unexpected signature", symbolName)
	}
	return load()
}

func (lm *LibraryManager) Version() (string, string, error) {
	symbolName := HookVersionFunctionName
	symbol, err := lm.p.Lookup(symbolName)
	if err != nil {
		return "", "", errors.Wrapf(err, "lookup for symbol: %s failed", symbolName)
	}
	versionFunction, ok := symbol.(HookVersionFunction)
	if !ok {
		return "", "", errors.Errorf("symbol %s has unexpected signature", symbolName)
	}
	program, version := versionFunction()
	return program, version, nil
}

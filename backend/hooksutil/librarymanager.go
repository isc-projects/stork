package hooksutil

import (
	"path"
	"plugin"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"isc.org/stork/hooks"
	storkutil "isc.org/stork/util"
)

// It is impossible to mock the `plugin.Plugin` struct directly. It's an
// interface that defines the same method as the plugin struct. It may be used
// to instantiate the library manager without a physical plugin file.
type pluginInterface interface {
	Lookup(string) (plugin.Symbol, error)
}

// Wrapper for a raw Go plugin to easier extraction of expected symbols
// (functions).
type LibraryManager struct {
	path string
	p    pluginInterface
}

// Opens a hook file and constructs the library manager object. Returns an
// error if the provided file isn't a valid Go plugin. It doesn't validate if
// the file is a valid Stork hook; the hook library will be created for any
// proper Go plugin.
func NewLibraryManager(path string) (*LibraryManager, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot open a plugin: %s", path)
	}

	return newLibraryManager(path, p), nil
}

// Internal constructor that accepts in-memory plugin (opened plugin
// or mock).
func newLibraryManager(path string, plugin pluginInterface) *LibraryManager {
	return &LibraryManager{path, plugin}
}

// Extracts and calls the CreateCLIFlags function of the Stork hook.
// Returns the CLI flags object or nil if hook doesn't require
// configuring. Returns error if the symbol is invalid but none if it doesn't
// exist.
func (lm *LibraryManager) CreateCLIFlags() (any, error) {
	symbolName := hooks.HookCreateCLIFlagsFunctionName
	symbol, err := lm.p.Lookup(symbolName)
	if err != nil {
		// The only possible error from the lookup function in Go 1.19 is:
		// 	errors.New("plugin: symbol " + symName + " not found in plugin " + p.pluginpath)
		// Source: tools/golang/go/src/plugin/plugin_dlopen.go:141
		// Date: 2023-06-22
		// Unfortunately, it doesn't have a type that can be checked.

		// The CreateCLIFlags member is optional. Return nil settings and
		// continue if it is missing.
		return nil, nil
	}

	createCLIFlagsFunction, ok := symbol.(hooks.HookCreateCLIFlagsFunction)
	if !ok {
		return nil, errors.Errorf("symbol %s has unexpected signature", symbolName)
	}

	cliFlagsInstance := createCLIFlagsFunction()
	if storkutil.IsNilPtr(cliFlagsInstance) {
		// The hook doesn't support configuration. Return nil settings
		// and continue.
		//
		// Nil pointer is a pointer but it isn't recognized as a struct pointer
		// by the below condition.
		return nil, nil
	}

	// Check the type of the returned value. It must be a pointer to structure.
	cliFlagsInstanceValue := reflect.ValueOf(cliFlagsInstance)

	if cliFlagsInstanceValue.Kind() != reflect.Pointer || cliFlagsInstanceValue.Elem().Kind() != reflect.Struct {
		return nil, errors.Errorf("returned cli flags object must be a pointer to struct")
	}
	return cliFlagsInstance, nil
}

// Extracts and calls the Load function of the Stork hook. Accepts the
// configured settings or nil if hook doesn't require configuration. Returns an
// error if the function is missing or fails. On success, it returns a callout
// carrier (an object with the callout specification implementations). The
// object also implements the Closer interface that must be called to unload
// the hook.
func (lm *LibraryManager) Load(settings hooks.HookSettings) (hooks.CalloutCarrier, error) {
	symbolName := hooks.HookLoadFunctionName
	symbol, err := lm.p.Lookup(symbolName)
	if err != nil {
		return nil, errors.Wrapf(err, "lookup for symbol: %s failed", symbolName)
	}

	load, ok := symbol.(hooks.HookLoadFunction)
	if !ok {
		return nil, errors.Errorf("symbol %s has unexpected signature", symbolName)
	}

	carrier, err := load(settings)
	err = errors.Wrap(err, "cannot load the hook")

	return carrier, err
}

// Extracts and calls the GetVersion function of the Stork hook. Returns an error if
// the function is missing or fails. The output contains the compatible
// application name (agent or server) and the expected Stork version.
func (lm *LibraryManager) GetVersion() (program string, version string, err error) {
	symbolName := hooks.HookGetVersionFunctionName
	symbol, err := lm.p.Lookup(symbolName)
	if err != nil {
		err = errors.Wrapf(err, "lookup for symbol: %s failed", symbolName)
		return
	}

	getVersionFunction, ok := symbol.(hooks.HookGetVersionFunction)
	if !ok {
		err = errors.Errorf("symbol %s has unexpected signature", symbolName)
		return
	}

	program, version = getVersionFunction()
	return
}

// Returns a path to the hook file.
func (lm *LibraryManager) GetPath() string {
	return lm.path
}

// Returns a hook filename without an extension.
func (lm *LibraryManager) GetName() string {
	return strings.TrimSuffix(path.Base(lm.path), path.Ext(lm.path))
}

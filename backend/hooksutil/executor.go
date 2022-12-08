package hooksutil

import (
	"reflect"

	"github.com/sirupsen/logrus"
	"isc.org/stork/hooks"
)

// Function that calls a specific callout point in the callout object.
type Caller = func(callout any)

// Manages all loaded hooks and allows to call their callout points.
// The caller may choose different calling strategies.
type HookExecutor struct {
	registeredCallouts map[reflect.Type][]hooks.Callout
}

// Constructs the hook executor using a list of supported callout types.
func NewHookExecutor(calloutTypes []reflect.Type) *HookExecutor {
	callouts := make(map[reflect.Type][]hooks.Callout, len(calloutTypes))
	for _, calloutType := range calloutTypes {
		if calloutType.Kind() != reflect.Interface {
			// It should never happen.
			// If you got this panic message, check if your callout types are
			// defined as follows:
			// reflect.TypeOf((*hooks.FooCallout)(nil)).Elem()
			// remember about:
			// 1. pointer (star *) before the callout type.
			// 2. .Elem() call at the end.
			panic("non-interface type passed")
		}
		callouts[calloutType] = make([]hooks.Callout, 0)
	}
	return &HookExecutor{
		registeredCallouts: callouts,
	}
}

// Registers a callout object in the hook executor. If the given type is
// unsupported, then it's silently ignored.
func (he *HookExecutor) registerCallout(callout hooks.Callout) {
	for calloutType, callouts := range he.registeredCallouts {
		if reflect.TypeOf(callout).Implements(calloutType) {
			he.registeredCallouts[calloutType] = append(callouts, callout)
		}
	}
}

// Unregisters all callout objects by calling their Close methods.
func (he *HookExecutor) unregisterAllCallouts() []error {
	var errs []error

	for _, callouts := range he.registeredCallouts {
		for _, callout := range callouts {
			err := callout.Close()
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	he.registeredCallouts = make(map[reflect.Type][]hooks.Callout)

	return errs
}

// Returns a slice of the supported callout types.
func (he *HookExecutor) GetSupportedCalloutTypes() []reflect.Type {
	supportedTypes := make([]reflect.Type, len(he.registeredCallouts))
	i := 0
	for t := range he.registeredCallouts {
		supportedTypes[i] = t
		i++
	}
	return supportedTypes
}

// Returns true if a given callout type is supported and has at least one
// callout object registered.
func (he *HookExecutor) HasRegistered(calloutType reflect.Type) bool {
	callouts, ok := he.registeredCallouts[calloutType]
	return ok && len(callouts) > 0
}

// Calls the specific callout point using the caller object.
// It can be used to monitor performance in the future.
func callCallout[TCallout hooks.Callout, TOutput any](callout TCallout, caller func(TCallout) TOutput) TOutput {
	return caller(callout)
}

// Calls the specific callout point for all callout objects sequentially, one by one.
func CallSequential[TCallout hooks.Callout, TOutput any](he *HookExecutor, caller func(TCallout) TOutput) []TOutput {
	t := reflect.TypeOf((*TCallout)(nil)).Elem()
	callouts, ok := he.registeredCallouts[t]
	if !ok {
		return nil
	}

	var results []TOutput
	for _, callout := range callouts {
		result := callCallout(callout.(TCallout), caller)
		results = append(results, result)
	}
	return results
}

// Calls the specific callout point from a first callout object if any was
// registered. It is dedicated to cases when only one hook with a given callout
// point is expected.
// Returns a default value if no callout point was called.
func CallSingle[TCallout hooks.Callout, TOutput any](he *HookExecutor, caller func(TCallout) TOutput) (output TOutput) {
	t := reflect.TypeOf((*TCallout)(nil)).Elem()
	callouts, ok := he.registeredCallouts[t]
	if !ok || len(callouts) == 0 {
		return
	} else if len(callouts) > 1 {
		logrus.
			WithField("callout", t.Name()).
			Warn("there are many registered callout objects but expected a single one")
	}
	return callCallout(callouts[0].(TCallout), caller)
}

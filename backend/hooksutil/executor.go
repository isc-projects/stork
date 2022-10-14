package hooksutil

import (
	"io"
	"reflect"

	"github.com/sirupsen/logrus"
)

// Function that calls a specific callout point in the callout object.
type Caller = func(callouts any)

// Manages all loaded hooks and allows to call their callout points.
// The caller may choose different calling strategies.
type HookExecutor struct {
	registeredCallouts map[reflect.Type][]any
}

// Constructs the hook executor using a list of supported callout types.
func NewHookExecutor(calloutTypes []reflect.Type) *HookExecutor {
	callouts := make(map[reflect.Type][]any, len(calloutTypes))
	for _, calloutType := range calloutTypes {
		if calloutType.Kind() != reflect.Interface {
			// It should never happen.
			// If you got this panic message, check if your callout types are
			// defined as follow:
			// reflect.TypeOf((*hooks.FooCallout)(nil)).Elem()
			// remember about:
			// 1. pointer (star *) before the callout type.
			// 2. .Elem() call at the end.
			panic("non-interface type passed")
		}
		callouts[calloutType] = make([]any, 0)
	}
	return &HookExecutor{
		registeredCallouts: callouts,
	}
}

// Registers a callout object in the hook executor. If the given type is
// unsupported, then it's silently ignored.
func (he *HookExecutor) RegisterCallouts(callouts any) {
	for calloutType, registeredCallouts := range he.registeredCallouts {
		if reflect.TypeOf(callouts).Implements(calloutType) {
			he.registeredCallouts[calloutType] = append(registeredCallouts, callouts)
		}
	}
}

// Unregisters all callout objects by calling their Close methods.
func (he *HookExecutor) UnregisterAllCallouts() []error {
	errs := []error{}

	for _, registeredCallouts := range he.registeredCallouts {
		for _, callout := range registeredCallouts {
			closer, ok := callout.(io.Closer)
			if !ok {
				continue
			}

			err := closer.Close()
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	he.registeredCallouts = make(map[reflect.Type][]any)

	return errs
}

// Returns all callout objects that implements a given callout type.
// If the callout type is not supported, returns false.
func (he *HookExecutor) getCallouts(calloutType reflect.Type) ([]any, bool) {
	callouts, ok := he.registeredCallouts[calloutType]
	return callouts, ok
}

// Returns true if a given callout type is supported and has at least one
// callout object registered.
func (he *HookExecutor) HasRegistered(calloutType reflect.Type) bool {
	callouts, ok := he.registeredCallouts[calloutType]
	return ok && len(callouts) > 0
}

// Calls the specific callout point using the caller object.
// It can be used to monitor performance in the future.
func callCallout[TCallout any, TOutput any](callout TCallout, caller func(TCallout) TOutput) TOutput {
	return caller(callout)
}

// Calls the specific callout point for all callout objects sequentially, one by one.
func CallSequential[TCallout any, TOutput any](he *HookExecutor, caller func(TCallout) TOutput) []TOutput {
	t := reflect.TypeOf((*TCallout)(nil)).Elem()
	allCallouts, ok := he.getCallouts(t)
	if !ok {
		return nil
	}

	var results []TOutput
	for _, callouts := range allCallouts {
		result := callCallout(callouts.(TCallout), caller)
		results = append(results, result)
	}
	return results
}

// Calls the specific callout point from a first callout object if any was
// registered. It is dedicated to cases when only one hook with a given callout
// point is expected.
// Returns a default value if no callout point was called.
func CallSingle[TCallout any, TOutput any](he *HookExecutor, caller func(TCallout) TOutput) (output TOutput) {
	t := reflect.TypeOf((*TCallout)(nil)).Elem()
	allCallouts, ok := he.getCallouts(t)
	if !ok || len(allCallouts) == 0 {
		return
	} else if len(allCallouts) > 1 {
		logrus.
			WithField("callout", t.Name()).
			Warn("there are many registered callouts but expected a single one")
	}
	return callCallout(allCallouts[0].(TCallout), caller)
}

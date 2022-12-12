package hooksutil

import (
	"reflect"

	"github.com/sirupsen/logrus"
	"isc.org/stork/hooks"
)

// Function that calls a specific callout in the callout carrier.
type Caller = func(carrier hooks.CalloutCarrier)

// Manages all loaded hooks and allows to call their callouts.
// The caller may choose different calling strategies.
type HookExecutor struct {
	registeredCarriers map[reflect.Type][]hooks.CalloutCarrier
}

// Constructs the hook executor using a list of supported callout carrier types.
func NewHookExecutor(calloutCarrierTypes []reflect.Type) *HookExecutor {
	carriers := make(map[reflect.Type][]hooks.CalloutCarrier, len(calloutCarrierTypes))
	for _, carrierType := range calloutCarrierTypes {
		if carrierType.Kind() != reflect.Interface {
			// It should never happen.
			// If you got this panic message, check if your callout types are
			// defined as follows:
			// reflect.TypeOf((*hooks.FooCallout)(nil)).Elem()
			// remember about:
			// 1. pointer (star *) before the callout type.
			// 2. .Elem() call at the end.
			panic("non-interface type passed")
		}
		carriers[carrierType] = make([]hooks.CalloutCarrier, 0)
	}
	return &HookExecutor{
		registeredCarriers: carriers,
	}
}

// Registers a callout carrier in the hook executor. If the given type is
// unsupported, then it's silently ignored.
func (he *HookExecutor) registerCalloutCarrier(carrier hooks.CalloutCarrier) {
	for carrierType, carriers := range he.registeredCarriers {
		if reflect.TypeOf(carrier).Implements(carrierType) {
			he.registeredCarriers[carrierType] = append(carriers, carrier)
		}
	}
}

// Unregisters all callout carriers by calling their Close methods.
func (he *HookExecutor) unregisterAllCalloutCarriers() []error {
	var errs []error

	for _, carriers := range he.registeredCarriers {
		for _, carrier := range carriers {
			err := carrier.Close()
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	he.registeredCarriers = make(map[reflect.Type][]hooks.CalloutCarrier)

	return errs
}

// Returns a slice of the supported callout carrier types.
func (he *HookExecutor) GetSupportedCalloutCarrierTypes() []reflect.Type {
	supportedTypes := make([]reflect.Type, len(he.registeredCarriers))
	i := 0
	for t := range he.registeredCarriers {
		supportedTypes[i] = t
		i++
	}
	return supportedTypes
}

// Returns true if a given callout carrier type is supported and has at least one
// callout carrier registered.
func (he *HookExecutor) HasRegistered(carrierType reflect.Type) bool {
	carriers, ok := he.registeredCarriers[carrierType]
	return ok && len(carriers) > 0
}

// Calls the specific callout using the caller object.
// It can be used to monitor performance in the future.
func callCallout[TCarrier hooks.CalloutCarrier, TOutput any](carrier TCarrier, caller func(TCarrier) TOutput) TOutput {
	return caller(carrier)
}

// Calls the specific callout for all callout carriers sequentially, one by one.
func CallSequential[TCarrier hooks.CalloutCarrier, TOutput any](he *HookExecutor, caller func(TCarrier) TOutput) []TOutput {
	t := reflect.TypeOf((*TCarrier)(nil)).Elem()
	carriers, ok := he.registeredCarriers[t]
	if !ok {
		return nil
	}

	var results []TOutput
	for _, carrier := range carriers {
		result := callCallout(carrier.(TCarrier), caller)
		results = append(results, result)
	}
	return results
}

// Calls the specific callout from a first callout carrier if any was
// registered. It is dedicated to cases when only one hook with a given callout
// is expected.
// Returns a default value if no callout was called.
func CallSingle[TCarrier hooks.CalloutCarrier, TOutput any](he *HookExecutor, caller func(TCarrier) TOutput) (output TOutput) {
	t := reflect.TypeOf((*TCarrier)(nil)).Elem()
	carriers, ok := he.registeredCarriers[t]
	if !ok || len(carriers) == 0 {
		return
	} else if len(carriers) > 1 {
		logrus.
			WithField("carrier", t.Name()).
			Warn("there are many registered callout carriers but expected a single one")
	}
	return callCallout(carriers[0].(TCarrier), caller)
}

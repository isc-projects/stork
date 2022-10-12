package hooksutil

import (
	"io"
	"reflect"

	"github.com/sirupsen/logrus"
)

type Caller = func(callouts any)

type HookExecutor struct {
	registeredCallouts map[reflect.Type][]any
}

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

func (he *HookExecutor) RegisterCallouts(callouts any) {
	for calloutType, registeredCallouts := range he.registeredCallouts {
		if reflect.TypeOf(callouts).Implements(calloutType) {
			he.registeredCallouts[calloutType] = append(registeredCallouts, callouts)
		} else {
			logrus.Warnf("unknown callouts: %s", reflect.TypeOf(calloutType).Name())
		}
	}
}

func (he *HookExecutor) UnregisterAllCallouts() {
	for _, registeredCallouts := range he.registeredCallouts {
		for _, callout := range registeredCallouts {
			if closer, ok := callout.(io.Closer); ok {
				closer.Close()
			}
		}
	}
	he.registeredCallouts = make(map[reflect.Type][]any)
}

func (he *HookExecutor) GetCallouts(calloutType reflect.Type) ([]any, bool) {
	callouts, ok := he.registeredCallouts[calloutType]
	return callouts, ok
}

func (he *HookExecutor) HasRegistered(calloutType reflect.Type) bool {
	callouts, ok := he.registeredCallouts[calloutType]
	return ok && len(callouts) > 0
}

// It can monitor performance.
func callCallout[TCallout any, TOutput any](callout TCallout, caller func(TCallout) TOutput) TOutput {
	return caller(callout)
}

func CallSequential[TCallout any, TOutput any](he *HookExecutor, caller func(TCallout) TOutput) []TOutput {
	t := reflect.TypeOf((*TCallout)(nil)).Elem()
	allCallouts, ok := he.GetCallouts(t)
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

func CallSingle[TCallout any, TOutput any](he *HookExecutor, caller func(TCallout) TOutput) (output TOutput) {
	t := reflect.TypeOf((*TCallout)(nil)).Elem()
	allCallouts, ok := he.GetCallouts(t)
	if !ok {
		return
	}

	if len(allCallouts) == 0 {
		return
	} else if len(allCallouts) > 1 {
		logrus.
			WithField("callout", t.Name()).
			Warn("there are many registered callouts but expected a single one")
	}
	return callCallout(allCallouts[0].(TCallout), caller)
}

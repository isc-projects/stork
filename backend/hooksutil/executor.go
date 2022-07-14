package hooksutil

import (
	"io"
	"reflect"

	"github.com/sirupsen/logrus"
)

type Caller = func(callouts interface{})

type HookExecutor struct {
	registeredCallouts map[reflect.Type][]interface{}
}

func NewHookExecutor(calloutTypes []reflect.Type) *HookExecutor {
	callouts := make(map[reflect.Type][]interface{}, len(calloutTypes))
	for _, calloutType := range calloutTypes {
		if calloutType.Kind() != reflect.Interface {
			// It should never happen.
			// If you got this panic message, check if your callout types are
			// definied as follow:
			// reflect.TypeOf((*hooks.FooCallout)(nil)).Elem()
			// remember about:
			// 1. pointer (star *) before the callout type.
			// 2. .Elem() call at the end.
			panic("non-interface type passed")
		}
		callouts[calloutType] = make([]interface{}, 0)
	}
	return &HookExecutor{
		registeredCallouts: callouts,
	}
}

func (he *HookExecutor) RegisterCallouts(callouts interface{}) {
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
	he.registeredCallouts = make(map[reflect.Type][]interface{})
}

func (he *HookExecutor) GetCallouts(calloutType reflect.Type) ([]interface{}, bool) {
	callouts, ok := he.registeredCallouts[calloutType]
	return callouts, ok
}

func (he *HookExecutor) HasRegistered(calloutType reflect.Type) bool {
	callouts, ok := he.registeredCallouts[calloutType]
	return ok && len(callouts) > 0
}

// It can monitor performance.
func callCallout[C interface{}, O interface{}](callout C, caller func(C) O) O {
	return caller(callout)
}

func CallSequential[C interface{}, O interface{}](he *HookExecutor, caller func(C) O) []O {
	t := reflect.TypeOf((*C)(nil)).Elem()
	allCallouts, ok := he.GetCallouts(t)
	if !ok {
		return nil
	}

	var results []O
	for _, callouts := range allCallouts {
		result := callCallout(callouts.(C), caller)
		results = append(results, result)
	}
	return results
}

func CallSingle[C interface{}, O interface{}](he *HookExecutor, caller func(C) O) (output O) {
	t := reflect.TypeOf((*C)(nil)).Elem()
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
	return callCallout(allCallouts[0].(C), caller)
}

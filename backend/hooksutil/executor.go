package hooksutil

import (
	"io"
	"reflect"

	"github.com/sirupsen/logrus"
)

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

func (he *HookExecutor) CallSequential(calloutType reflect.Type, caller func(callouts interface{})) {
	for _, callouts := range he.registeredCallouts[calloutType] {
		caller(callouts)
	}
}

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

// It can monitor performance
func (he *HookExecutor) callCallout(callout interface{}, caller Caller) {
	caller(callout)
}

func (he *HookExecutor) CallSequential(calloutType reflect.Type, caller Caller) {
	for _, callouts := range he.registeredCallouts[calloutType] {
		he.callCallout(callouts, caller)
	}
}

func (he *HookExecutor) CallSingle(calloutType reflect.Type, caller Caller) {
	if len(he.registeredCallouts[calloutType]) == 0 {
		return
	} else if len(he.registeredCallouts[calloutType]) > 1 {
		logrus.
			WithField("callout", calloutType.Name()).
			Warn("there are many registered callouts but expected a single one")
	}
	he.callCallout(he.registeredCallouts[calloutType][0], caller)
}

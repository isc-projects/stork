package serverhooks

import (
	"reflect"

	"isc.org/stork/hooks"
	"isc.org/stork/hooks/server/authenticationcallout"
	"isc.org/stork/hooksutil"
	dbmodel "isc.org/stork/server/database/model"
	"isc.org/stork/server/gen/restapi/operations/users"
)

var (
	authenticationCalloutType reflect.Type = reflect.TypeOf((*authenticationcallout.AuthenticationCallout)(nil)).Elem()
)

func newHookExecutor() *hooksutil.HookExecutor {
	executor := hooksutil.NewHookExecutor([]reflect.Type{
		authenticationCalloutType,
	})
	return executor
}

type HookManager struct {
	executor *hooksutil.HookExecutor
}

// Interface checks
var _ authenticationcallout.AuthenticationCallout = (*HookManager)(nil)

func newHookManager(executor *hooksutil.HookExecutor) *HookManager {
	return &HookManager{
		executor: executor,
	}
}

func NewHookManagerFromDirectory(directory string) *HookManager {
	allCallouts := hooksutil.LoadAllHooks(hooks.HookProgramServer, directory)
	return NewHookManagerFromCallouts(allCallouts)
}

func NewHookManagerFromCallouts(allCallouts []interface{}) *HookManager {
	executor := newHookExecutor()
	for _, callouts := range allCallouts {
		executor.RegisterCallouts(callouts)
	}
	return newHookManager(executor)
}

func (hm *HookManager) Close() {
	hm.executor.UnregisterAllCallouts()
}

func (hm *HookManager) GetAuthenticationDetails() *authenticationcallout.AuthenticationMethodDetails {
	return hm.executor.
}

func (hm *HookManager) Authenticate(params users.CreateSessionParams) (*dbmodel.SystemUser, error) {

}

func (hm *HookManager) Unauthenticate(user *dbmodel.SystemUser) error {

}

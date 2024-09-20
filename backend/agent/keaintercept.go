package agent

import (
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	agentapi "isc.org/stork/api"
	keactrl "isc.org/stork/appctrl/kea"
)

// Structure containing a pointer to the callback function registered in
// in the Kea interceptor and associated with one of the Kea commands.
// The callback is invoked when the given command is received by the
// agent and after it is forwarded to Kea.
type keaInterceptorHandler struct {
	callback func(*StorkAgent, *keactrl.Response) error
}

// Structure holding a collection of handlers/callabacks to be invoked
// for a given Kea command.
type keaInterceptorTarget struct {
	// List of callbacks to be invoked for a command.
	handlers []*keaInterceptorHandler
}

// The Kea interceptor is a generic mechanism for dispatching calls to
// the registered callback functions when agent forwards a given command
// to the Kea server.
type keaInterceptor struct {
	mutex *sync.Mutex
	// Holds a list of async callbacks to be invoked for a given command.
	asyncTargets map[keactrl.CommandName]*keaInterceptorTarget
	// Holds a list of sync callbacks to be invoked for a given command.
	syncTargets map[keactrl.CommandName]*keaInterceptorTarget
}

// Creates new Kea interceptor instance.
func newKeaInterceptor() *keaInterceptor {
	interceptor := &keaInterceptor{
		mutex: new(sync.Mutex),
	}
	interceptor.asyncTargets = make(map[keactrl.CommandName]*keaInterceptorTarget)
	interceptor.syncTargets = make(map[keactrl.CommandName]*keaInterceptorTarget)
	return interceptor
}

// Registers an asynchronous callback function and associates it with a given command.
// It is possible to register multiple callbacks for the same command.
func (i *keaInterceptor) registerAsync(callback func(*StorkAgent, *keactrl.Response) error, commandName keactrl.CommandName) {
	var (
		target *keaInterceptorTarget
		ok     bool
	)

	// Make sure we don't collide with asyncHandle calls.
	i.mutex.Lock()
	defer i.mutex.Unlock()

	// Check if the target for the given command already exists.
	target, ok = i.asyncTargets[commandName]
	if !ok {
		// This is the first time we register callback for this command.
		// Let's create the target instance.
		target = &keaInterceptorTarget{}
		i.asyncTargets[commandName] = target
	}
	// Create the handler from the callback and associate it with the
	// given target/command.
	h := &keaInterceptorHandler{
		callback: callback,
	}
	target.handlers = append(target.handlers, h)
}

// Registers a synchronous callback function and associates it with a given command.
// It is possible to register multiple callbacks for the same command.
func (i *keaInterceptor) registerSync(callback func(*StorkAgent, *keactrl.Response) error, commandName keactrl.CommandName) {
	var (
		target *keaInterceptorTarget
		ok     bool
	)

	// Check if the target for the given command already exists.
	target, ok = i.syncTargets[commandName]
	if !ok {
		// This is the first time we register callback for this command.
		// Let's create the target instance.
		target = &keaInterceptorTarget{}
		i.syncTargets[commandName] = target
	}
	// Create the handler from the callback and associate it with the
	// given target/command.
	h := &keaInterceptorHandler{
		callback: callback,
	}
	target.handlers = append(target.handlers, h)
}

// Triggers invocation of all sync callbacks registered for the given command. The
// callback is invoked separately for each daemon which responded to the command.
// The result of the callbacks may to affect the response forwarded to the Stork Server.
// Synchronous handler is executed before asynchronous one.
func (i *keaInterceptor) syncHandle(agent *StorkAgent, request *agentapi.KeaRequest, response []byte) ([]byte, error) {
	changedResponse, err := i.handle(i.syncTargets, agent, request, response)
	err = errors.WithMessage(err, "Failed to execute synchronous handlers")
	return changedResponse, err
}

// Triggers invocation of all async callbacks registered for the given command. The
// callback is invoked separately for each daemon which responded to the command.
// This function should be invoked in the goroutine as it invokes the handlers
// which can be run independently from the agent. The agent may send back the
// response to the server while these callbacks are invoked. The result of the
// callbacks do not affect the response forwarded to the Stork Server.
func (i *keaInterceptor) asyncHandle(agent *StorkAgent, request *agentapi.KeaRequest, response []byte) {
	// We don't want to run the handlers concurrently in case they update the same
	// data structures. Also, we want to avoid registration of handlers while we're
	// here.
	i.mutex.Lock()
	defer i.mutex.Unlock()

	_, err := i.handle(i.asyncTargets, agent, request, response)
	if err != nil {
		log.Errorf("Failed to execute asynchronous handler: %+v", err)
	}
}

// Common part of asynchronous and synchronous handlers. Returns the serialized
// response after modifications performed by callbacks or error.
func (i *keaInterceptor) handle(targets map[keactrl.CommandName]*keaInterceptorTarget, agent *StorkAgent, request *agentapi.KeaRequest, response []byte) ([]byte, error) {
	// Parse the request to get the command name and service.
	command, err := keactrl.NewCommandFromJSON(request.Request)
	if err != nil {
		err = errors.WithMessage(err, "Failed to parse Kea command")
		return nil, err
	}

	// Check if there is any handler registered for this command.
	target, ok := targets[command.Command]
	if !ok {
		return response, nil
	}

	// Parse the response. It will be passed to the callback so as the callback
	// can "do something" with it.
	var parsedResponse keactrl.ResponseList
	err = keactrl.UnmarshalResponseList(command, response, &parsedResponse)
	if err != nil {
		err = errors.WithMessagef(err, "Failed to parse Kea responses for command %s", command.Command)
		return nil, err
	}

	// Check what daemons the callbacks need to be invoked for.
	var daemons []keactrl.DaemonName
	if len(command.Daemons) == 0 {
		daemons = append(daemons, "ca")
	} else {
		daemons = command.Daemons
	}
	// Invoke callbacks for each handler registered for this command.
	for i := range target.handlers {
		// Invoke the handler for each daemon.
		for j := range daemons {
			if j < len(parsedResponse) {
				callback := target.handlers[i].callback
				if callback != nil {
					err = callback(agent, &parsedResponse[j])
					if err != nil {
						err = errors.WithMessagef(err, "Callback returned an error for command %s", command.Command)
						return nil, err
					}
				}
			}
		}
	}

	// Serialize response after modifications.
	response, err = keactrl.MarshalResponseList(parsedResponse)
	if err != nil {
		err = errors.WithMessagef(err, "Failed to marshal changed responses for command %s", command.Command)
		return nil, err
	}
	return response, nil
}

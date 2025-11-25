package agent

import (
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	keactrl "isc.org/stork/daemonctrl/kea"
)

// Structure containing a pointer to the callback function registered in
// in the Kea interceptor and associated with one of the Kea commands.
// The callback is invoked when the given command is received by the
// agent and after it is forwarded to Kea.
type keaInterceptorHandler struct {
	callback func(*StorkAgent, *keactrl.Response) error
}

// Structure holding a collection of handlers/callbacks to be invoked
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
func (i *keaInterceptor) syncHandle(agent *StorkAgent, request keactrl.SerializableCommand, response keactrl.Response) (keactrl.Response, error) {
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
func (i *keaInterceptor) asyncHandle(agent *StorkAgent, request keactrl.SerializableCommand, response keactrl.Response) {
	// We don't want to run the handlers concurrently in case they update the same
	// data structures. Also, we want to avoid registration of handlers while we're
	// here.
	i.mutex.Lock()
	defer i.mutex.Unlock()

	_, err := i.handle(i.asyncTargets, agent, request, response)
	if err != nil {
		log.WithError(err).Error("Failed to execute asynchronous handler")
	}
}

// Common part of asynchronous and synchronous handlers. Returns the serialized
// response after modifications performed by callbacks or error.
func (i *keaInterceptor) handle(targets map[keactrl.CommandName]*keaInterceptorTarget, agent *StorkAgent, command keactrl.SerializableCommand, response keactrl.Response) (keactrl.Response, error) {
	// Check if there is any handler registered for this command.
	target, ok := targets[command.GetCommand()]
	if !ok {
		return response, nil
	}

	// Check what daemons the callbacks need to be invoked for.
	daemons := command.GetDaemonsList()
	switch len(daemons) {
	case 0:
		// No daemon specified, this field is required by Stork agent.
		return keactrl.Response{}, errors.Errorf("no daemon specified in the command %s", command.GetCommand())
	case 1:
		// Only one daemon specified, so we can proceed.
	default:
		// More than one daemon specified. This is not supported.
		return keactrl.Response{}, errors.Errorf("multiple daemons specified in the command %s", command.GetCommand())
	}

	// Invoke callbacks for each handler registered for this command.
	processedResponse := response
	for i := range target.handlers {
		// Invoke the handler.
		callback := target.handlers[i].callback
		if callback != nil {
			err := callback(agent, &processedResponse)
			if err != nil {
				err = errors.WithMessagef(err, "Callback returned an error for command %s", command.GetCommand())
				return keactrl.Response{}, err
			}
		}
	}
	return processedResponse, nil
}

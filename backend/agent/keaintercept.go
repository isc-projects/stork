package agent

import (
	"sync"

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
	// Holds a list of callbacks to be invoked for a given command.
	asyncTargets map[string]*keaInterceptorTarget
}

// Creates new Kea interceptor instance.
func newKeaInterceptor() *keaInterceptor {
	interceptor := &keaInterceptor{
		mutex: new(sync.Mutex),
	}
	interceptor.asyncTargets = make(map[string]*keaInterceptorTarget)
	return interceptor
}

// Registers a callback function and associates it with a given command.
// It is possible to register multiple callbacks for the same command.
func (i *keaInterceptor) register(callback func(*StorkAgent, *keactrl.Response) error, commandName string) {
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

// Triggers invocation of all callbacks registered for the given command. the
// callback is invoked separately for each daemon which responded to the command.
// This function should be invoked in the goroutine as it invokes the handlers
// which can be run independently from the agent. The agent may send back the
// response to the server while these callbacks are invoked. The result of the
// callbacks do not affect the response forwarded to the Stork server.
func (i *keaInterceptor) asyncHandle(agent *StorkAgent, request *agentapi.KeaRequest, response []byte) {
	// Parse the request to get the command name and service.
	command, err := keactrl.NewCommandFromJSON(request.Request)
	if err != nil {
		log.Errorf("failed parse Kea command while invoking asynchronous handlers: %+v", err)
		return
	}

	// Check if there is any handler registered for this command.
	var (
		target *keaInterceptorTarget
		ok     bool
	)

	// We don't want to run the handlers concurrently in case they update the same
	// data structures. Also, we want to avoid registration of handlers while we're
	// here.
	i.mutex.Lock()
	defer i.mutex.Unlock()

	target, ok = i.asyncTargets[command.Command]
	if !ok {
		return
	}

	// Parse the response. It will be passed to the callback so as the callback
	// can "do something" with it.
	var parsedResponse keactrl.ResponseList
	err = keactrl.UnmarshalResponseList(command, response, &parsedResponse)
	if err != nil {
		log.Errorf("failed to parse Kea responses while invoking asynchronous handlers for command %s: %+v",
			command.Command, err)
		return
	}

	// Check what daemons the callbacks need to be invoked for.
	var daemons *keactrl.Daemons
	if command.Daemons == nil || len(*command.Daemons) == 0 {
		daemons, _ = keactrl.NewDaemons("ca")
	} else {
		daemons = command.Daemons
	}
	daemonsList := daemons.List()

	// Invoke callbacks for each handler registered for this command.
	for i := range target.handlers {
		// Invoke the handler for each daemon.
		for j := range daemonsList {
			if j < len(parsedResponse) {
				callback := target.handlers[i].callback
				if callback != nil {
					err = callback(agent, &parsedResponse[j])
					if err != nil {
						log.Warnf("asynchronous callback returned an error for command %s: %+v",
							command.Command, err)
					}
				}
			}
		}
	}
}

package eventcenter

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/go-pg/pg/v9"
	log "github.com/sirupsen/logrus"

	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
)

// An interface to EventCenter
type EventCenter interface {
	AddInfoEvent(text string, objects ...interface{})
	AddWarningEvent(text string, objects ...interface{})
	AddErrorEvent(text string, objects ...interface{})
	AddEvent(event *dbmodel.Event)
	Shutdown()
	ServeHTTP(w http.ResponseWriter, req *http.Request)
}

// EventCenter. It has channel for receiving events
// and a SSE broker for dispatching events to subscribers.
type eventCenter struct {
	db     *dbops.PgDB
	done   chan bool
	wg     *sync.WaitGroup
	events chan *dbmodel.Event

	sseBroker *SSEBroker
}

// Create new EventCenter object.
func NewEventCenter(db *pg.DB) EventCenter {
	ec := &eventCenter{
		db:        db,
		done:      make(chan bool),
		wg:        &sync.WaitGroup{},
		events:    make(chan *dbmodel.Event),
		sseBroker: NewSSEBroker(),
	}
	ec.wg.Add(1)
	go ec.mainLoop()

	log.Printf("Started EventCenter")
	return ec
}

// Add an event on info level to EventCenter. It takes event text and relating objects.
// The event is stored in database and dispatched to subscribers.
func (ec *eventCenter) AddInfoEvent(text string, objects ...interface{}) {
	ec.addEvent(dbmodel.EvInfo, text, objects...)
}

// Add an event on warning level to EventCenter. It takes event text and relating objects.
// The event is stored in database and dispatched to subscribers.
func (ec *eventCenter) AddWarningEvent(text string, objects ...interface{}) {
	ec.addEvent(dbmodel.EvWarning, text, objects...)
}

// Add an event on error level to EventCenter. It takes event text and relating objects.
// The event is stored in database and dispatched to subscribers.
func (ec *eventCenter) AddErrorEvent(text string, objects ...interface{}) {
	ec.addEvent(dbmodel.EvError, text, objects...)
}

// Create an event without passing it to EventCenter. It can be added later using
// AddEvent method of EventCenter. It takes event level, text and relating objects.
func CreateEvent(level int, text string, objects ...interface{}) *dbmodel.Event {
	relations := &dbmodel.Relations{}
	var details string
	for _, obj := range objects {
		if d, ok := obj.(*dbmodel.Daemon); ok {
			text = strings.Replace(text, "{daemon}", daemonTag(d), -1)
			relations.DaemonID = d.ID
		} else if app, ok := obj.(*dbmodel.App); ok {
			text = strings.Replace(text, "{app}", appTag(app), -1)
			relations.AppID = app.ID
		} else if m, ok := obj.(*dbmodel.Machine); ok {
			text = strings.Replace(text, "{machine}", machineTag(m), -1)
			relations.MachineID = m.ID
		} else if s, ok := obj.(*dbmodel.Subnet); ok {
			text = strings.Replace(text, "{subnet}", subnetTag(s), -1)
			relations.SubnetID = s.ID
		} else if s, ok := obj.(string); ok {
			if len(s) > 0 {
				details = s
			}
		} else {
			log.Warnf("unknown object passed to CreateEvent: %v", obj)
		}
	}
	e := &dbmodel.Event{
		Text:      text,
		Level:     level,
		Relations: relations,
		Details:   details,
	}
	return e
}

func (ec *eventCenter) addEvent(level int, text string, objects ...interface{}) {
	e := CreateEvent(level, text, objects...)
	ec.AddEvent(e)
}

// Add event object to EventCenter. The event object can be prepared
// manually or using CreateEvent function. The event is stored in
// database and dispatched to subscribers.
func (ec *eventCenter) AddEvent(event *dbmodel.Event) {
	log.Printf("event '%s'", event.Text)
	ec.events <- event
}

// Terminate the EventCenter main loop.
func (ec *eventCenter) Shutdown() {
	log.Printf("Stopping EventCenter")
	ec.done <- true
	ec.wg.Wait()
	log.Printf("Stopped EventCenter")
}

// A main loop of EventCenter. It receives events via channel, stores
// them into database and dispatches them to subscribers using SSE broker.
func (ec *eventCenter) mainLoop() {
	defer ec.wg.Done()
	for {
		select {
		// wait for done signal from shutdown function
		case <-ec.done:
			return
		// get events from channel
		case event := <-ec.events:
			err := dbmodel.AddEvent(ec.db, event)
			if err != nil {
				log.Errorf("problem with adding event to db: %+v", err)
				continue
			}
			ec.sseBroker.dispatchEvent(event)
		}
	}
}

// Forward SSE requests to SSE Broker.
func (ec *eventCenter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ec.sseBroker.ServeHTTP(w, req)
}

// Prepare a tag describing a daemon.
func daemonTag(daemon *dbmodel.Daemon) string {
	tag := fmt.Sprintf("<daemon id=\"%d\" name=\"%s\" appId=\"%d\" appType=\"%s\">", daemon.ID, daemon.Name, daemon.AppID, daemon.App.Type)
	return tag
}

// Prepare a tag describing an app.
func appTag(app *dbmodel.App) string {
	tag := fmt.Sprintf("<app id=\"%d\" type=\"%s\" version=\"%s\">",
		app.ID, app.Type, app.Meta.Version)
	return tag
}

// Prepare a tag describing a machine.
func machineTag(machine *dbmodel.Machine) string {
	tag := fmt.Sprintf("<machine id=\"%d\" address=\"%s\" hostname=\"%s\">",
		machine.ID, machine.Address, machine.State.Hostname)
	return tag
}

// Prepare a tag describing a subnet.
func subnetTag(subnet *dbmodel.Subnet) string {
	tag := fmt.Sprintf("<subnet id=\"%d\" prefix=\"%s\">",
		subnet.ID, subnet.Prefix)
	return tag
}

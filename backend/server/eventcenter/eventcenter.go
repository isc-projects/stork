package eventcenter

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/go-pg/pg/v10"
	log "github.com/sirupsen/logrus"
	dbops "isc.org/stork/server/database"
	dbmodel "isc.org/stork/server/database/model"
)

// An interface to EventCenter.
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
		sseBroker: NewSSEBroker(db),
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
func CreateEvent(level dbmodel.EventLevel, text string, objects ...interface{}) *dbmodel.Event {
	relations := &dbmodel.Relations{}
	streams := []dbmodel.SSEStream{}
	var details string
	for _, obj := range objects {
		switch entity := obj.(type) {
		case dbmodel.DaemonTag:
			text = strings.ReplaceAll(text, "{daemon}", daemonTag(entity))
			relations.DaemonID = entity.GetID()
			relations.AppID = entity.GetAppID()
			if machineID := entity.GetMachineID(); machineID != nil {
				relations.MachineID = *machineID
			}

		case dbmodel.AppTag:
			text = strings.ReplaceAll(text, "{app}", appTag(entity))
			relations.AppID = entity.GetID()
			relations.MachineID = entity.GetMachineID()

		case dbmodel.MachineTag:
			text = strings.ReplaceAll(text, "{machine}", machineTag(entity))
			relations.MachineID = entity.GetID()

		case *dbmodel.Subnet:
			text = strings.ReplaceAll(text, "{subnet}", subnetTag(entity))
			relations.SubnetID = entity.ID

		case *dbmodel.SystemUser:
			text = strings.ReplaceAll(text, "{user}", userTag(entity))
			relations.UserID = int64(entity.ID)

		case dbmodel.SSEStream:
			streams = append(streams, entity)

		case string:
			s := obj.(string)
			if len(s) > 0 {
				details = s
			}

		case error:
			s := entity.Error()
			if len(s) > 0 {
				details = s
			}

		case []error:
			var errors []string
			for _, err := range entity {
				if err == nil {
					// The error list are used when some subsequent or parallel
					// operations are performed and some of them may fail. If
					// the error is nil, it means that the operation was
					// successful, so we must skip it.
					continue
				}
				errors = append(errors, err.Error())
			}
			details = strings.Join(errors, "; ")

		default:
			log.Warnf("Unknown object passed to CreateEvent: %v", obj)
		}
	}
	e := &dbmodel.Event{
		Text:       text,
		Level:      level,
		Relations:  relations,
		Details:    details,
		SSEStreams: streams,
	}
	return e
}

func (ec *eventCenter) addEvent(level dbmodel.EventLevel, text string, objects ...interface{}) {
	e := CreateEvent(level, text, objects...)
	ec.AddEvent(e)
}

// Add event object to EventCenter. The event object can be prepared
// manually or using CreateEvent function. The event is stored in
// database and dispatched to subscribers.
func (ec *eventCenter) AddEvent(event *dbmodel.Event) {
	log.Printf("Event '%s'", event.Text)
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
			ec.sseBroker.shutdown()
			return
		// get events from channel
		case event := <-ec.events:
			err := dbmodel.AddEvent(ec.db, event)
			if err != nil {
				log.Errorf("Problem adding event to db: %+v", err)
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
func daemonTag(daemon dbmodel.DaemonTag) string {
	tag := fmt.Sprintf("<daemon id=\"%d\" name=\"%s\" appId=\"%d\" appType=\"%s\">", daemon.GetID(), daemon.GetName(), daemon.GetAppID(), daemon.GetAppType())
	return tag
}

// Prepare a tag describing an app.
func appTag(app dbmodel.AppTag) string {
	tag := fmt.Sprintf("<app id=\"%d\" name=\"%s\" type=\"%s\" version=\"%s\">",
		app.GetID(), app.GetName(), app.GetType(), app.GetVersion())
	return tag
}

// Prepare a tag describing a machine.
func machineTag(machine dbmodel.MachineTag) string {
	tag := fmt.Sprintf("<machine id=\"%d\" address=\"%s\" hostname=\"%s\">",
		machine.GetID(), machine.GetAddress(), machine.GetHostname())
	return tag
}

// Prepare a tag describing a subnet.
func subnetTag(subnet *dbmodel.Subnet) string {
	tag := fmt.Sprintf("<subnet id=\"%d\" prefix=\"%s\">",
		subnet.ID, subnet.Prefix)
	return tag
}

// Prepare a tag describing a user.
func userTag(user *dbmodel.SystemUser) string {
	tag := fmt.Sprintf("<user id=\"%d\" login=\"%s\" email=\"%s\">",
		user.ID, user.Login, user.Email)
	return tag
}

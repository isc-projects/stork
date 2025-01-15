package dnsop

import (
	"context"

	"github.com/go-pg/pg/v10"
	"isc.org/stork/appdata/bind9stats"
	agentcomm "isc.org/stork/server/agentcomm"
	dbmodel "isc.org/stork/server/database/model"
)

type ManagerAccessors interface {
	// Returns an instance of the database handler used by the manager.
	GetDB() *pg.DB
	// Returns an interface to the agents the manager communicates with.
	GetConnectedAgents() agentcomm.ConnectedAgents
}

type Manager struct {
	db     *pg.DB
	agents agentcomm.ConnectedAgents
}

func NewManager(owner ManagerAccessors) *Manager {
	return &Manager{
		db:     owner.GetDB(),
		agents: owner.GetConnectedAgents(),
	}
}

func (manager *Manager) FetchZones() error {
	apps, err := dbmodel.GetAppsByType(manager.db, dbmodel.AppTypeBind9)
	if err != nil {
		return err
	}
	for _, app := range apps {
		batch := dbmodel.NewBatchUpsert[*dbmodel.Zone](manager.db, 1000, dbmodel.AddZones)
		ctx := context.Background()
		manager.agents.ReceiveZones(ctx, &app, nil, func(zone *bind9stats.ExtendedZone, err error) {
			dbZone := dbmodel.Zone{
				Name: zone.Name(),
			}
			err = batch.Add(&dbZone, false)
			if err != nil {
				return
			}
		})
	}
	return nil
}

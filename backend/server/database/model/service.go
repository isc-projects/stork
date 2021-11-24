package dbmodel

import (
	"errors"
	"time"

	"github.com/go-pg/pg/v9"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// A structure reflecting service SQL table. This table holds
// generic information about the service such as ID, service name
// and service type.
type BaseService struct {
	tableName   struct{} `pg:"service"` //nolint:unused,structcheck
	ID          int64
	Name        string
	ServiceType string
	CreatedAt   time.Time

	Daemons []*Daemon `pg:"many2many:daemon_to_service,fk:service_id,joinFK:daemon_id"`
}

// A structure reflecting a daemon_to_service SQL table which associates
// daemons with services in many to many relationship.
type DaemonToService struct {
	DaemonID  int64 `pg:",pk"`
	ServiceID int64 `pg:",pk"`
}

// A structure holding HA specific information about the service. It
// reflects the ha_service table which extends the service table with
// High Availability specific information. It is embedded in the
// Service structure.
type BaseHAService struct {
	tableName                   struct{} `pg:"ha_service"` //nolint:unused,structcheck
	ID                          int64
	ServiceID                   int64
	HAType                      string
	HAMode                      string
	PrimaryID                   int64
	SecondaryID                 int64
	BackupID                    []int64 `pg:",array"`
	PrimaryStatusCollectedAt    time.Time
	SecondaryStatusCollectedAt  time.Time
	PrimaryLastState            string
	SecondaryLastState          string
	PrimaryLastScopes           []string `pg:",array"`
	SecondaryLastScopes         []string `pg:",array"`
	PrimaryReachable            bool
	SecondaryReachable          bool
	PrimaryLastFailoverAt       time.Time
	SecondaryLastFailoverAt     time.Time
	PrimaryCommInterrupted      *bool `pg:",use_zero"`
	SecondaryCommInterrupted    *bool `pg:",use_zero"`
	PrimaryConnectingClients    int64
	SecondaryConnectingClients  int64
	PrimaryUnackedClients       int64
	SecondaryUnackedClients     int64
	PrimaryUnackedClientsLeft   int64
	SecondaryUnackedClientsLeft int64
	PrimaryAnalyzedPackets      int64
	SecondaryAnalyzedPackets    int64
}

// A structure reflecting all SQL tables holding information about the
// services of various types. It embeds the BaseService structure which
// holds the basic information about the service. It also embeds
// HAService structure which holds High Availability specific information
// if the service is of the HA type. It is nil if the service is not
// of the HA type. This structure is to be extended with additional
// structures as more service types are defined.
type Service struct {
	BaseService
	HAService *BaseHAService
}

// Associates daemons with a service in the database. The dbIface parameter
// can be of pg.DB or pg.Tx type. In the first case, this function will
// start new transaction for adding all association. In the second case
// the already started transaction will be used.
func addServiceDaemons(dbIface interface{}, service *BaseService) (err error) {
	if len(service.Daemons) == 0 {
		return nil
	}

	var assocs []DaemonToService

	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		err = pkgerrors.WithMessage(err, "problem with starting transaction for adding daemons to a service")
	}
	defer rollback()

	// If there are any daemons to be associated with the service, let's iterate over them
	// and insert into the daemon_to_service table.
	for _, d := range service.Daemons {
		assocs = append(assocs, DaemonToService{
			DaemonID:  d.ID,
			ServiceID: service.ID,
		})
	}

	_, err = tx.Model(&assocs).OnConflict("DO NOTHING").Insert()

	// If we have started the transaction in this function we also have to commit the
	// changes.
	err = commit()
	if err != nil {
		err = pkgerrors.WithMessage(err, "problem with committing associations of daemon with the service")
	}

	return err
}

// Associate a daemon with the service having a specified id.
func AddDaemonToService(dbIface interface{}, serviceID int64, daemon *Daemon) error {
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with starting transaction for associating a daemon %d with the service %d",
			daemon.ID, serviceID)
		return err
	}
	defer rollback()

	service := &BaseService{
		ID: serviceID,
	}
	service.Daemons = append(service.Daemons, daemon)
	err = addServiceDaemons(tx, service)
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with associating a daemon having id %d with service %d",
			daemon.ID, serviceID)
		return err
	}

	err = commit()
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with committing transaction associating daemon %d with the service %d",
			daemon.ID, service.ID)
	}
	return err
}

// Dissociate a daemon from the service having a specified id.
// The first returned value indicates if any row was removed from the
// daemon_to_service table.
func DeleteDaemonFromService(db *pg.DB, serviceID, daemonID int64) (bool, error) {
	as := &DaemonToService{
		DaemonID:  daemonID,
		ServiceID: serviceID,
	}
	rows, err := db.Model(as).WherePK().Delete()
	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrapf(err, "problem with deleting a daemon with id %d from the service %d",
			daemonID, serviceID)
		return false, err
	}
	return rows.RowsAffected() > 0, nil
}

// Adds new service to the database and associates the daemons with it.
// This operation is performed in a transaction. There are several SQL tables
// involved in this operation: service, daemon_to_service and optionally ha_service.
func AddService(dbIface interface{}, service *Service) error {
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with starting transaction for adding new service")
		return err
	}
	defer rollback()

	// Insert generic information into the service table.
	_, err = tx.Model(service).Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with adding new service")
		return err
	}

	// Add associations of the daemons with the service.
	err = addServiceDaemons(tx, &service.BaseService)
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with associating daemons with a new service")
		return err
	}

	// If this is HA service, let's add extra HA specific information into the
	// ha_service table.
	if service.HAService != nil {
		err = AddHAService(tx, service.ID, service.HAService)
		if err != nil {
			return err
		}
	}

	// All ok, let's commit the changes.
	err = commit()
	if err != nil {
		err = pkgerrors.WithMessage(err, "problem with committing new service into the database")
	}

	return err
}

// Adds information about HA service and associate it with the given service ID.
func AddHAService(dbIface interface{}, serviceID int64, haService *BaseHAService) error {
	haService.ServiceID = serviceID

	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		return err
	}
	defer rollback()

	_, err = tx.Model(haService).Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with adding new HA service to the database")
		return err
	}

	err = commit()
	if err != nil {
		err = pkgerrors.WithMessage(err, "problem with committing new HA service into the database")
	}
	return err
}

// Updates basic information about the service. It only affects the contents of the
// service table in the database.
func UpdateBaseService(dbIface interface{}, service *BaseService) error {
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		return err
	}
	defer rollback()

	err = tx.Update(service)
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with updating base service with id %d", service.ID)
		return err
	}

	err = commit()
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with committing base service %d information after update",
			service.ID)
	}
	return err
}

// Updates HA specific information for a service. It only affects the contents of
// the ha_service table.
func UpdateBaseHAService(dbIface interface{}, service *BaseHAService) error {
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		return err
	}
	defer rollback()

	err = tx.Update(service)
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with updating the HA information for service with id %d",
			service.ServiceID)
		return err
	}

	err = commit()
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with committing HA information for service with id %d after update",
			service.ServiceID)
	}
	return err
}

// Updates basic and detailed information about the service.
func UpdateService(dbIface interface{}, service *Service) error {
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		return err
	}
	defer rollback()

	err = UpdateBaseService(tx, &service.BaseService)
	if err != nil {
		return err
	}

	if service.HAService != nil {
		if service.HAService.ID == 0 {
			err = AddHAService(tx, service.ID, service.HAService)
		} else {
			err = UpdateBaseHAService(tx, service.HAService)
		}
		if err != nil {
			return err
		}
	}

	err = commit()
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem with committing service with id %d after update", service.ID)
	}
	return err
}

// Fetches a service from the database for a given service id.
func GetDetailedService(db *dbops.PgDB, serviceID int64) (*Service, error) {
	service := &Service{}
	err := db.Model(service).
		Relation("HAService").
		Relation("Daemons.KeaDaemon.KeaDHCPDaemon").
		Relation("Daemons.App").
		Where("service.id = ?", serviceID).
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem with getting a service with id %d", serviceID)
		return nil, err
	}
	return service, err
}

// Fetches all services to which the given app belongs.
func GetDetailedServicesByAppID(db *dbops.PgDB, appID int64) ([]Service, error) {
	var services []Service

	err := db.Model(&services).
		Join("INNER JOIN daemon_to_service AS dtos ON dtos.service_id = service.id").
		Join("INNER JOIN daemon AS d ON d.id = dtos.daemon_id").
		Join("INNER JOIN app AS a ON d.app_id = a.ID").
		Relation("HAService").
		Relation("Daemons.KeaDaemon.KeaDHCPDaemon").
		Relation("Daemons.App").
		Relation("Daemons.App.AccessPoints").
		Where("app_id = ?", appID).
		OrderExpr("service.id ASC").
		Select()

	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrapf(err, "problem with getting services for app id %d", appID)
		return services, err
	}

	return services, nil
}

// Fetches all services from the database.
func GetDetailedAllServices(dbi dbops.DBI) ([]Service, error) {
	var services []Service

	err := dbi.Model(&services).
		Relation("HAService").
		Relation("Daemons.KeaDaemon.KeaDHCPDaemon").
		Relation("Daemons.App").
		OrderExpr("id ASC").
		Select()

	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrapf(err, "problem with getting all services")
		return services, err
	}
	return services, nil
}

// Deletes the service and all associations of this service with the
// applications. The applications are not removed.
func DeleteService(db *dbops.PgDB, serviceID int64) error {
	service := &Service{
		BaseService: BaseService{
			ID: serviceID,
		},
	}
	_, err := db.Model(service).WherePK().Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem with deleting the service having id %d", serviceID)
	}
	return err
}

// Iterates over the services and commits them to the database. It also associates them
// with the specified daemon.
func CommitServicesIntoDB(dbIface interface{}, services []Service, daemon *Daemon) error {
	// Begin transaction.
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		return err
	}
	defer rollback()

	for i := range services {
		if services[i].ID == 0 {
			err = AddService(tx, &services[i])
		} else {
			err = UpdateService(tx, &services[i])
		}
		if err != nil {
			err = pkgerrors.WithMessagef(err, "problem with committing services into the database")
			return err
		}
		// Try to associate the app with the service. If the association already
		// exists this is no-op.
		err = AddDaemonToService(tx, services[i].ID, daemon)
		if err != nil {
			err = pkgerrors.WithMessagef(err, "problem with associating detected service %d with daemon having id %d",
				services[i].ID, daemon.ID)
			return err
		}
	}

	err = commit()
	if err != nil {
		err = pkgerrors.WithMessage(err, "problem with committing services into the database")
	}
	return err
}

// Checks if the service is new, i.e. hasn't yet been inserted into
// the database.
func (s Service) IsNew() bool {
	return s.ID == 0
}

// Returns the High Availability state for the given service and daemon.
func (s Service) GetDaemonHAState(daemonID int64) string {
	if s.HAService == nil {
		return ""
	}
	if s.HAService.PrimaryID == daemonID {
		return s.HAService.PrimaryLastState
	}
	if s.HAService.SecondaryID == daemonID {
		return s.HAService.SecondaryLastState
	}
	for _, id := range s.HAService.BackupID {
		if id == daemonID {
			return "backup"
		}
	}
	return ""
}

// Returns last failover time of the given daemon's partner, i.e. the
// time when the given daemon was considered offline for the last time
// by the HA peer. The partner may have crashed but it may also be
// the case that the communication with it was interrupted even though
// it was online.
func (s Service) GetPartnerHAFailureTime(daemonID int64) (failureTime time.Time) {
	if s.HAService == nil {
		return failureTime
	}
	if s.HAService.PrimaryID == daemonID {
		failureTime = s.HAService.SecondaryLastFailoverAt
	} else if s.HAService.SecondaryID == daemonID {
		failureTime = s.HAService.PrimaryLastFailoverAt
	}
	return failureTime
}

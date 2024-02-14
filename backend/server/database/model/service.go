package dbmodel

import (
	"context"
	"errors"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

// Registers M:N SQL relations defined in this file.
func init() {
	orm.RegisterTable((*DaemonToService)(nil))
}

// A structure reflecting service SQL table. This table holds
// generic information about the service such as ID, service name
// and service type.
type BaseService struct {
	tableName   struct{} `pg:"service"` //nolint:unused
	ID          int64
	Name        string
	ServiceType string
	CreatedAt   time.Time

	Daemons []*Daemon `pg:"many2many:daemon_to_service,fk:service_id,join_fk:daemon_id"`
}

// A structure reflecting a daemon_to_service SQL table which associates
// daemons with services in many to many relationship.
type DaemonToService struct {
	DaemonID  int64 `pg:",pk"`
	ServiceID int64 `pg:",pk"`
}

// High Availability mode.
type HAMode = string

// High Availability type.
type HAType = string

// High Availability state.
type HAState = string

// Valid values for the HA-related enums.
const (
	HAModeHotStandby    HAMode = "hot-standby"
	HAModePassiveBackup HAMode = "passive-backup"
	HAModeLoadBalancing HAMode = "load-balancing"

	HATypeDhcp4 HAType = "dhcp4"
	HATypeDhcp6 HAType = "dhcp6"

	HAStateNone                  HAState = ""
	HAStateBackup                HAState = "backup"
	HAStateCommunicationRecovery HAState = "communication-recovery"
	HAStateHotStandby            HAState = "hot-standby"
	HAStateLoadBalancing         HAState = "load-balancing"
	HAStateInMaintenance         HAState = "in-maintenance"
	HAStatePartnerDown           HAState = "partner-down"
	HAStatePartnerInMaintenance  HAState = "partner-in-maintenance"
	HAStatePassiveBackup         HAState = "passive-backup"
	HAStateReady                 HAState = "ready"
	HAStateSyncing               HAState = "syncing"
	HAStateTerminated            HAState = "terminated"
	HAStateWaiting               HAState = "waiting"
	HAStateUnavailable           HAState = "unavailable"
)

// A structure holding HA specific information about the service. It
// reflects the ha_service table which extends the service table with
// High Availability specific information. It is embedded in the
// Service structure.
type BaseHAService struct {
	tableName                   struct{} `pg:"ha_service"` //nolint:unused
	ID                          int64
	ServiceID                   int64
	HAType                      HAType
	HAMode                      HAMode
	Relationship                string
	PrimaryID                   int64
	SecondaryID                 int64
	BackupID                    []int64 `pg:",array"`
	PrimaryStatusCollectedAt    time.Time
	SecondaryStatusCollectedAt  time.Time
	PrimaryLastState            HAState
	SecondaryLastState          HAState
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
	HAService *BaseHAService `pg:"rel:belongs-to"`
}

// Associates daemons with a service in the database. The first parameter
// is a pointer to the current transaction.
func addServiceDaemons(tx *pg.Tx, service *BaseService) (err error) {
	if len(service.Daemons) == 0 {
		return nil
	}

	var assocs []DaemonToService

	// If there are any daemons to be associated with the service, let's iterate over them
	// and insert into the daemon_to_service table.
	for _, d := range service.Daemons {
		assocs = append(assocs, DaemonToService{
			DaemonID:  d.ID,
			ServiceID: service.ID,
		})
	}

	_, err = tx.Model(&assocs).OnConflict("DO NOTHING").Insert()

	return err
}

// Associate a daemon with the service in a transaction.
func addDaemonToService(tx *pg.Tx, serviceID int64, daemon *Daemon) error {
	service := &BaseService{
		ID: serviceID,
	}
	service.Daemons = append(service.Daemons, daemon)
	err := addServiceDaemons(tx, service)
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem associating the daemon with ID %d with service %d",
			daemon.ID, serviceID)
	}
	return err
}

// Associate a daemon with the service. It begins a new transaction when
// dbi has a *pg.DB type or uses an existing transaction when dbi has a
// *pg.Tx type.
func AddDaemonToService(dbi dbops.DBI, serviceID int64, daemon *Daemon) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addDaemonToService(tx, serviceID, daemon)
		})
	}
	return addDaemonToService(dbi.(*pg.Tx), serviceID, daemon)
}

// Dissociate a daemon from the service having a specified id.
// The first returned value indicates if any row was removed from the
// daemon_to_service table.
func DeleteDaemonFromService(dbi dbops.DBI, serviceID, daemonID int64) (bool, error) {
	as := &DaemonToService{
		DaemonID:  daemonID,
		ServiceID: serviceID,
	}
	rows, err := dbi.Model(as).WherePK().Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem deleting the daemon with ID %d from the service %d",
			daemonID, serviceID)
		return false, err
	}
	return rows.RowsAffected() > 0, nil
}

// Dissociates a daemon from the services. The first returned value indicates
// how many rows have been removed from the daemon_to_service table.
func DeleteDaemonFromServices(dbi dbops.DBI, daemonID int64) (int64, error) {
	result, err := dbi.Model((*DaemonToService)(nil)).
		Where("daemon_id = ?", daemonID).
		Delete()
	if err != nil && !errors.Is(err, pg.ErrNoRows) {
		err = pkgerrors.Wrapf(err, "problem deleting daemon %d from services", daemonID)
		return 0, err
	}
	return int64(result.RowsAffected()), nil
}

// Adds new service to the database and associates the daemons with it
// in a transaction.
func addService(tx *pg.Tx, service *Service) error {
	// Insert generic information into the service table.
	_, err := tx.Model(service).Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem adding new service")
		return err
	}

	// Add associations of the daemons with the service.
	err = addServiceDaemons(tx, &service.BaseService)
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem associating daemons with a new service")
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
	return nil
}

// Adds new service to the database and associates the daemons with it.
// It begins a new transaction when dbi has a *pg.DB type or uses an
// existing transaction when dbi has a *pg.Tx type. There are several SQL
// tables involved in this operation: service, daemon_to_service and
// optionally ha_service.
func AddService(dbi dbops.DBI, service *Service) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return addService(tx, service)
		})
	}
	return addService(dbi.(*pg.Tx), service)
}

// Adds information about HA service and associates it with the given service ID.
func AddHAService(dbi dbops.DBI, serviceID int64, haService *BaseHAService) error {
	haService.ServiceID = serviceID
	_, err := dbi.Model(haService).Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem adding new HA service to the database")
	}
	return err
}

// Updates basic information about the service. It only affects the contents of the
// service table in the database.
func UpdateBaseService(dbi dbops.DBI, service *BaseService) error {
	result, err := dbi.Model(service).WherePK().ExcludeColumn("created_at").Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem updating base service with ID %d", service.ID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "service with ID %d does not exist", service.ID)
	}
	return err
}

// Updates HA specific information for a service. It only affects the contents of
// the ha_service table.
func UpdateBaseHAService(dbi dbops.DBI, service *BaseHAService) error {
	result, err := dbi.Model(service).WherePK().Update()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem updating HA information for service with ID %d",
			service.ServiceID)
		return err
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "service with ID %d does not exist", service.ServiceID)
	}
	return err
}

// Updates basic and detailed information about the service in a
// transaction.
func updateService(tx *pg.Tx, service *Service) error {
	err := UpdateBaseService(tx, &service.BaseService)
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
	return nil
}

// Updates basic and detailed information about the service. It begins a new
// transaction when dbi has a *pg.DB type or uses an existing transaction when
// dbi has a *pg.Tx type.
func UpdateService(dbi dbops.DBI, service *Service) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return updateService(tx, service)
		})
	}
	return updateService(dbi.(*pg.Tx), service)
}

// Fetches a service from the database for a given service id.
func GetDetailedService(dbi dbops.DBI, serviceID int64) (*Service, error) {
	service := &Service{}
	err := dbi.Model(service).
		Relation("HAService").
		Relation("Daemons.KeaDaemon.KeaDHCPDaemon").
		Relation("Daemons.App").
		Where("service.id = ?", serviceID).
		Select()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return nil, nil
		}
		err = pkgerrors.Wrapf(err, "problem getting the service with ID %d", serviceID)
		return nil, err
	}
	return service, err
}

// Fetches all services to which the given app belongs.
func GetDetailedServicesByAppID(dbi dbops.DBI, appID int64) ([]Service, error) {
	var services []Service

	err := dbi.Model(&services).
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
		err = pkgerrors.Wrapf(err, "problem getting services for app ID %d", appID)
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
		err = pkgerrors.Wrapf(err, "problem getting all services")
		return services, err
	}
	return services, nil
}

// Deletes the service and all associations of this service with the
// applications. The applications are not removed.
func DeleteService(dbi dbops.DBI, serviceID int64) error {
	service := &Service{
		BaseService: BaseService{
			ID: serviceID,
		},
	}
	result, err := dbi.Model(service).WherePK().Delete()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem deleting the service having ID %d", serviceID)
	} else if result.RowsAffected() <= 0 {
		err = pkgerrors.Wrapf(ErrNotExists, "service with ID %d does not exist", serviceID)
	}
	return err
}

// Iterates over the services and commits them to the database. It also associates them
// with the specified daemon. The services are committed in a transaction.
func commitServicesIntoDB(tx *pg.Tx, services []Service, daemon *Daemon) error {
	var err error
	for i := range services {
		if services[i].ID == 0 {
			err = AddService(tx, &services[i])
		} else {
			err = UpdateService(tx, &services[i])
		}
		if err != nil {
			err = pkgerrors.WithMessagef(err, "problem committing services into the database")
			return err
		}
		// Try to associate the app with the service. If the association already
		// exists this is no-op.
		err = AddDaemonToService(tx, services[i].ID, daemon)
		if err != nil {
			err = pkgerrors.WithMessagef(err, "problem associating detected service %d with daemon of ID %d",
				services[i].ID, daemon.ID)
			return err
		}
	}
	return nil
}

// Iterates over the services and commits them to the database. It also associates them
// with the specified daemon. It begins a new transaction when dbi has a *pg.DB type
// or uses an existing transaction when dbi has a *pg.Tx type.
func CommitServicesIntoDB(dbi dbops.DBI, services []Service, daemon *Daemon) error {
	if db, ok := dbi.(*pg.DB); ok {
		return db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			return commitServicesIntoDB(tx, services, daemon)
		})
	}
	return commitServicesIntoDB(dbi.(*pg.Tx), services, daemon)
}

// Checks if the service is new, i.e. hasn't yet been inserted into
// the database.
func (s Service) IsNew() bool {
	return s.ID == 0
}

// Returns the High Availability state for the given service and daemon.
func (s Service) GetDaemonHAState(daemonID int64) HAState {
	if s.HAService == nil {
		return HAStateNone
	}
	if s.HAService.PrimaryID == daemonID {
		return s.HAService.PrimaryLastState
	}
	if s.HAService.SecondaryID == daemonID {
		return s.HAService.SecondaryLastState
	}
	for _, id := range s.HAService.BackupID {
		if id == daemonID {
			return HAStateBackup
		}
	}
	return HAStateNone
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

// Checks if the HA state is operational.
func isOperationalHAState(state HAState) bool {
	switch state {
	case HAStateHotStandby, HAStateLoadBalancing, HAStatePartnerDown,
		HAStatePartnerInMaintenance, HAStateReady:
		return true
	default:
		return false
	}
}

// Returns the HA daemons that don't allocate leases independently (depend on
// another server or don't allocate at all).
func GetPassiveHADaemonIDs(db dbops.DBI) ([]int64, error) {
	services, err := GetDetailedAllServices(db)
	if err != nil {
		return nil, err
	}

	passiveHADaemons := make([]int64, 0)

	for _, service := range services {
		if service.HAService == nil {
			continue
		}

		// Backups never actively allocate leases.
		passiveHADaemons = append(passiveHADaemons, service.HAService.BackupID...)

		// The server is operational if it is reachable and has an operational state.
		isPrimaryOperational := isOperationalHAState(service.HAService.PrimaryLastState)
		isPrimaryOperational = isPrimaryOperational && service.HAService.PrimaryReachable

		isSecondaryOperational := isOperationalHAState(service.HAService.SecondaryLastState)
		isSecondaryOperational = isSecondaryOperational && service.HAService.SecondaryReachable

		if isPrimaryOperational || !isSecondaryOperational {
			passiveHADaemons = append(passiveHADaemons, service.HAService.SecondaryID)
		} else {
			passiveHADaemons = append(passiveHADaemons, service.HAService.PrimaryID)
		}
	}

	return passiveHADaemons, nil
}

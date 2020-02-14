package dbmodel

import (
	"github.com/go-pg/pg/v9"
	"github.com/pkg/errors"

	dbops "isc.org/stork/server/database"

	"time"
)

// A structure reflecting service SQL table. This table holds
// generic information about the service such as ID, service name
// and service type.
type BaseService struct {
	tableName   struct{} `pg:"service"` //nolint:unused,structcheck
	ID          int64
	Name        string
	ServiceType string
	Created     time.Time

	Apps []*App `pg:"many2many:app_to_service,fk:service_id,joinFK:app_id"`
}

// A structure reflecting an app_to_service SQL table which associates
// applications with services in many to many relationship.
type AppToService struct {
	AppID     int64 `pg:",pk"`
	ServiceID int64 `pg:",pk"`
}

// A structure holding HA specific information about the service. It
// reflects the ha_service table which extends the service table with
// High Availability specific information. It is embedded in the
// Service structure.
type BaseHAService struct {
	tableName           struct{} `pg:"ha_service"` //nolint:unused,structcheck
	ID                  int64
	ServiceID           int64
	HAType              string
	HAMode              string
	PrimaryID           int64
	SecondaryID         int64
	BackupID            []int64 `pg:",array"`
	PrimaryStatusTime   time.Time
	SecondaryStatusTime time.Time
	PrimaryLastState    string
	SecondaryLastState  string
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

// Associates apps with a service in the database. The dbIface parameter
// can be of pg.DB or pg.Tx type. In the first case, this function will
// start new transaction for adding all association. In the second case
// the already started transaction will be used.
func addServiceApps(dbIface interface{}, service *BaseService) (err error) {
	if len(service.Apps) == 0 {
		return nil
	}

	var assocs []AppToService

	var tx *pg.Tx
	db, ok := dbIface.(*pg.DB)
	if ok {
		// Database object provided rather than the transaction. Let's
		// create new transaction.
		tx, err = db.Begin()
		if err != nil {
			err = errors.Wrapf(err, "problem with starting transaction for adding apps to a service")
		}
		defer func() {
			_ = tx.Rollback()
		}()
	} else {
		// Transaction has been started by the caller, so let's just
		// use it.
		tx, ok = dbIface.(*pg.Tx)
		if !ok {
			err = errors.New("unsupported type of the database transaction object provided")
			return err
		}
	}

	// If there are any apps to be associated with the service, let's iterate over them
	// and insert into the app_to_service table.
	for _, a := range service.Apps {
		assocs = append(assocs, AppToService{
			AppID:     a.ID,
			ServiceID: service.ID,
		})
	}

	_, err = tx.Model(&assocs).OnConflict("DO NOTHING").Insert()

	// If we have started the transaction in this function we also have to commit the
	// changes.
	if db != nil {
		err = tx.Commit()
		if err != nil {
			err = errors.Wrapf(err, "problem with committing associations of application with the service")
		}
	}

	return err
}

// Associate an application with the service having a specified id.
func AddAppToService(db *pg.DB, serviceID int64, app *App) error {
	service := &BaseService{
		ID: serviceID,
	}
	service.Apps = append(service.Apps, app)
	err := addServiceApps(db, service)

	if err != nil {
		err = errors.Wrapf(err, "problem with associating an app having id %d with service %d",
			app.ID, serviceID)
	}
	return err
}

// Dissociate an application from the service having a specified id.
// The first returned value indicates if any row was removed from the
// app_to_service table.
func DeleteAppFromService(db *pg.DB, serviceID, appID int64) (bool, error) {
	as := &AppToService{
		AppID:     appID,
		ServiceID: serviceID,
	}
	rows, err := db.Model(as).WherePK().Delete()
	if err != nil && err != pg.ErrNoRows {
		err = errors.Wrapf(err, "problem with deleting an app with id %d from the service %d",
			appID, serviceID)
		return false, err
	}
	return rows.RowsAffected() > 0, nil
}

// Adds new service to the database and associates the applications with it.
// This operation is performed in a transaction. There are several SQL tables
// involved in this operation: service, app_to_service and optionally ha_service.
func AddService(db *dbops.PgDB, service *Service) error {
	tx, err := db.Begin()
	if err != nil {
		err = errors.Wrapf(err, "problem with starting transaction for adding new service")
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Insert generic information into the service table.
	_, err = tx.Model(service).Insert()
	if err != nil {
		err = errors.Wrapf(err, "problem with adding new service")
		return err
	}

	// Add associations of the apps with the service.
	err = addServiceApps(tx, &service.BaseService)
	if err != nil {
		err = errors.Wrapf(err, "problem with associating apps with a new service")
		return err
	}

	// If this is HA service, let's add extra HA specific information into the
	// ha_service table.
	if service.HAService != nil {
		service.HAService.ServiceID = service.ID
		_, err = tx.Model(service.HAService).Insert()
		if err != nil {
			err = errors.Wrapf(err, "problem with adding new High Availability service")
			return err
		}
	}

	// All ok, let's commit the changes.
	err = tx.Commit()
	if err != nil {
		err = errors.Wrapf(err, "problem with committing new service into the database")
	}

	return err
}

// Updates basic information about the service. It only affects the contents of the
// service table in the database.
func UpdateBaseService(db *dbops.PgDB, service *BaseService) error {
	err := db.Update(service)
	if err != nil {
		err = errors.Wrapf(err, "problem with updating a service with id %d", service.ID)
	}
	return err
}

// Updates HA specific information for a service. It only affects the contents of
// the ha_service table.
func UpdateBaseHAService(db *dbops.PgDB, service *BaseHAService) error {
	err := db.Update(service)
	if err != nil {
		err = errors.Wrapf(err, "problem with updating the HA configuration for service with id %d",
			service.ServiceID)
	}
	return err
}

// Fetches a service from the database for a given service id.
func GetDetailedService(db *dbops.PgDB, serviceID int64) (*Service, error) {
	service := &Service{}
	err := db.Model(service).
		Relation("HAService").
		Relation("Apps").
		Where("service.id = ?", serviceID).
		Select()

	if err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		err = errors.Wrapf(err, "problem with getting a service with id %d", serviceID)
		return nil, err
	}
	return service, err
}

// Fetches all services to which the given app belongs.
func GetDetailedServicesByAppID(db *dbops.PgDB, appID int64) ([]Service, error) {
	var services []Service

	err := db.Model(&services).
		Join("INNER JOIN app_to_service AS atos ON atos.service_id = service.id").
		Relation("HAService").
		Relation("Apps").
		Where("atos.app_id = ?", appID).
		OrderExpr("service.id ASC").
		Select()

	if err != nil && err != pg.ErrNoRows {
		err = errors.Wrapf(err, "problem with getting services for app id %d", appID)
		return services, err
	}
	return services, nil
}

// Fetches all services which include applications that operate on the specified
// control address and port. This is useful for detecting that an application,
// perhaps the one that is being now added, belongs to the existing service.
// In particular, the Kea application which belongs to the HA setup includes
// the URLs of the partners. These URLs can be used to associate the applications
// with the services.
func GetDetailedServicesByAppCtrlAddressPort(db *dbops.PgDB, ctrlAddress string, ctrlPort int64) ([]Service, error) {
	var services []Service

	err := db.Model(&services).
		DistinctOn("service.id").
		Join("INNER JOIN app_to_service AS atos ON atos.service_id = service.id").
		Join("INNER JOIN app AS a ON a.id = atos.app_id").
		Relation("HAService").
		Relation("Apps").
		Where("a.ctrl_address = ?", ctrlAddress).
		Where("a.ctrl_port = ?", ctrlPort).
		OrderExpr("service.id ASC").
		Select()

	if err != nil && err != pg.ErrNoRows {
		err = errors.Wrapf(err, "problem with getting services for control address: %s and port: %d",
			ctrlAddress, ctrlPort)
		return services, err
	}
	return services, nil
}

// Fetches all services from the database.
func GetDetailedAllServices(db *dbops.PgDB) ([]Service, error) {
	var services []Service

	err := db.Model(&services).
		Relation("HAService").
		Relation("Apps").
		OrderExpr("id ASC").
		Select()

	if err != nil && err != pg.ErrNoRows {
		err = errors.Wrapf(err, "problem with getting all services")
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
		err = errors.Wrapf(err, "problem with deleting the service having id %d", serviceID)
	}
	return err
}

// Checks if the service is new, i.e. hasn't yet been inserted into
// the database.
func (s Service) IsNew() bool {
	return s.ID == 0
}

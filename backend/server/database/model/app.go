package dbmodel

import (
	"context"
	"errors"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	pkgerrors "github.com/pkg/errors"
	"isc.org/stork/datamodel"
	dbops "isc.org/stork/server/database"
)

// Identifier of the relations between the app and other tables.
type AppRelation string

// Names of the app table relations. They must be valid in the go-pg sense.
const (
	AppRelationAccessPoints      = "AccessPoints"
	AppRelationMachine           = "Machine"
	AppRelationDaemons           = "Daemons"
	AppRelationKeaDaemons        = "Daemons.KeaDaemon"
	AppRelationKeaDHCPDaemons    = "Daemons.KeaDaemon.KeaDHCPDaemon"
	AppRelationBind9Daemons      = "Daemons.Bind9Daemon"
	AppRelationDaemonsLogTargets = "Daemons.LogTargets"
)

// A short for datamodel.AppType.
type AppType = datamodel.AppType

// Currently supported types are: "kea" and "bind9".
const (
	AppTypeKea   = datamodel.AppTypeKea
	AppTypeBind9 = datamodel.AppTypeBind9
)

// Part of app table in database that describes metadata of app. In DB it is stored as JSONB.
type AppMeta struct {
	Version         string
	ExtendedVersion string
}

// Represents an app held in app table in the database.
type App struct {
	ID        int64
	CreatedAt time.Time
	MachineID int64
	Machine   *Machine `pg:"rel:has-one"`
	Type      AppType  // currently supported types are: "kea" and "bind9"
	Active    bool
	Meta      AppMeta
	Name      string

	AccessPoints []*AccessPoint `pg:"rel:has-many"`

	Daemons []*Daemon `pg:"rel:has-many"`
}

// AppTag is an interface implemented by the dbmodel.App exposing functions
// to create events referencing apps.
type AppTag interface {
	GetID() int64
	GetName() string
	GetType() AppType
	GetVersion() string
	GetMachineID() int64
}

// updateAppAccessPoints updates the associated application access points into
// the database.
func updateAppAccessPoints(tx *pg.Tx, app *App, update bool) (err error) {
	if update {
		// First delete any access points previously associated with the app.
		types := []string{}
		for _, point := range app.AccessPoints {
			types = append(types, point.Type)
		}
		if len(types) > 0 {
			q := tx.Model((*AccessPoint)(nil))
			q = q.Where("app_id = ?", app.ID)
			q = q.Where("type NOT IN (?)", pg.In(types))
			_, err = q.Delete()
			if err != nil {
				return pkgerrors.Wrapf(err, "problem removing access points from app %d", app.ID)
			}
		}
	}

	// If there are any access points associated with the app,
	// iterate over them and insert/update into the access_points table.
	for _, point := range app.AccessPoints {
		point.AppID = app.ID
		point.MachineID = app.MachineID
		if update {
			_, err = tx.Model(point).OnConflict("(app_id, type) DO UPDATE").Insert()
		} else {
			_, err = tx.Model(point).Insert()
		}
		if err != nil {
			return pkgerrors.Wrapf(err, "problem adding access point to app %d: %v", app.ID, point)
		}
	}
	return nil
}

// Adds or updates daemons for an app. If any daemons already exist for the app,
// they are removed and the new daemons will be added instead. Return list of
// added daemons, deleted daemons and error if occurred.
func updateAppDaemons(tx *pg.Tx, app *App) ([]*Daemon, []*Daemon, error) {
	// Delete the existing daemons in database but not present in
	// app.Daemons list. The app.Daemons list contains new or
	// updated list of daemons. So, any daemons associated with
	// this app in the database but missing from the app.Daemons
	// list are going to be deleted from the database.
	ids := []int64{}
	for _, d := range app.Daemons {
		if d.ID > 0 {
			ids = append(ids, d.ID)
		}
	}
	q := tx.Model((*Daemon)(nil)).
		Where("daemon.app_id = ?", app.ID)
	if len(ids) > 0 {
		q = q.Where("daemon.id NOT IN (?)", pg.In(ids))
	}
	var deletedDaemons []*Daemon
	_, err := q.Returning("*").Delete(&deletedDaemons)
	if err != nil {
		return nil, nil, pkgerrors.Wrapf(err, "problem deleting daemons for an updated app %d", app.ID)
	}

	// Add updated daemons.
	var addedDaemons []*Daemon
	for i := range app.Daemons {
		daemon := app.Daemons[i]

		// Make sure the inserted daemon references the app.
		daemon.AppID = app.ID
		if daemon.ID == 0 {
			// Add the new entry to the daemon table.
			_, err = tx.Model(daemon).Insert()
			addedDaemons = append(addedDaemons, daemon)
			if err != nil {
				return nil, nil, pkgerrors.Wrapf(err, "problem inserting daemon in app %d: %v", app.ID, daemon)
			}
		} else {
			result, err := tx.Model(daemon).WherePK().Update()
			if err != nil {
				return nil, nil, pkgerrors.Wrapf(err, "problem updating daemon in app %d: %v", app.ID, daemon)
			} else if result.RowsAffected() <= 0 {
				return nil, nil, pkgerrors.Wrapf(ErrNotExists, "daemon with ID %d does not exist", daemon.ID)
			}
		}

		if daemon.KeaDaemon != nil {
			// Make sure that the kea_daemon references the daemon.
			daemon.KeaDaemon.DaemonID = daemon.ID
			err = upsertInTransaction(tx, daemon.KeaDaemon.ID, daemon.KeaDaemon)
			if err != nil {
				return nil, nil, pkgerrors.Wrapf(err, "problem upserting Kea daemon to app %d: %v",
					app.ID, daemon.KeaDaemon)
			}

			if daemon.KeaDaemon.KeaDHCPDaemon != nil {
				// Make sure that the kea_dhcp_daemon references the kea_daemon.
				daemon.KeaDaemon.KeaDHCPDaemon.KeaDaemonID = daemon.KeaDaemon.ID
				err = upsertInTransaction(tx, daemon.KeaDaemon.KeaDHCPDaemon.ID, daemon.KeaDaemon.KeaDHCPDaemon)
				if err != nil {
					return nil, nil, pkgerrors.Wrapf(err, "problem upserting Kea DHCP daemon to app %d: %v",
						app.ID, daemon.KeaDaemon.KeaDHCPDaemon)
				}
			}
		} else if daemon.Bind9Daemon != nil {
			// Make sure that the bind9_daemon references the daemon.
			daemon.Bind9Daemon.DaemonID = daemon.ID
			err = upsertInTransaction(tx, daemon.Bind9Daemon.ID, daemon.Bind9Daemon)
			if err != nil {
				return nil, nil, pkgerrors.Wrapf(err, "problem upserting BIND 9 daemon to app %d: %v",
					app.ID, daemon.Bind9Daemon)
			}
		}

		// Identify and delete the log targets that no longer exist for the daemon.
		ids = []int64{}
		for _, t := range daemon.LogTargets {
			if t.ID > 0 {
				ids = append(ids, t.ID)
			}
		}
		q := tx.Model((*LogTarget)(nil)).
			Where("log_target.daemon_id = ?", daemon.ID)
		if len(ids) > 0 {
			q = q.Where("log_target.id NOT IN (?)", pg.In(ids))
		}
		_, err := q.Delete()
		if err != nil {
			return nil, nil, pkgerrors.Wrapf(err, "problem deleting log targets for updated daemon %d",
				daemon.ID)
		}

		// Insert or update log targets.
		for i := range daemon.LogTargets {
			// If the log target has no id yet, it means that it is not yet
			// present in the database and should be inserted. Otherwise,
			// it is updated.
			if daemon.LogTargets[i].ID == 0 {
				// Make sure that the inserted log target is linked with the
				// daemon.
				daemon.LogTargets[i].DaemonID = daemon.ID
				_, err = tx.Model(daemon.LogTargets[i]).Insert()
				if err != nil {
					return nil, nil, pkgerrors.Wrapf(err, "problem inserting log target %s to daemon %d: %v",
						daemon.LogTargets[i].Output, daemon.ID, daemon)
				}
			} else {
				result, err := tx.Model(daemon.LogTargets[i]).WherePK().ExcludeColumn("created_at").Update()
				if err != nil {
					return nil, nil, pkgerrors.Wrapf(err, "problem updating log target %s in daemon %d: %v",
						daemon.LogTargets[i].Output, daemon.ID, daemon)
				} else if result.RowsAffected() <= 0 {
					return nil, nil, pkgerrors.Wrapf(ErrNotExists, "log target with ID %d does not exist",
						daemon.LogTargets[i].ID)
				}
			}
		}
	}
	return addedDaemons, deletedDaemons, nil
}

// Adds application into the database in a transaction.
func addApp(tx *pg.Tx, app *App) ([]*Daemon, error) {
	_, err := tx.Model(app).Insert()
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem inserting app %v", app)
	}
	addedDaemons, deletedDaemons, err := updateAppDaemons(tx, app)
	if err != nil {
		return nil, pkgerrors.WithMessage(err, "problem inserting daemons for a new app")
	}
	if len(deletedDaemons) > 0 {
		return nil, pkgerrors.Errorf("problem deleting daemons for a new app")
	}
	// Add access points.
	err = updateAppAccessPoints(tx, app, false)
	if err != nil {
		return nil, pkgerrors.WithMessagef(err, "problem adding access points to app: %+v", app)
	}
	return addedDaemons, nil
}

// Adds application into the database. It begins a new transaction when
// dbi has a *pg.DB type or uses an existing transaction when dbi has
// a *pg.Tx type.
func AddApp(dbi dbops.DBI, app *App) (addedDaemons []*Daemon, err error) {
	if db, ok := dbi.(*pg.DB); ok {
		err = db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			addedDaemons, err = addApp(tx, app)
			return err
		})
		return
	}
	addedDaemons, err = addApp(dbi.(*pg.Tx), app)
	return
}

// Updates application in the database in a transaction.
func updateApp(tx *pg.Tx, app *App) (addedDaemons []*Daemon, deletedDaemons []*Daemon, err error) {
	result, err := tx.Model(app).WherePK().ExcludeColumn("created_at").Update()
	if err != nil {
		return nil, nil, pkgerrors.Wrapf(err, "problem updating app %v", app)
	} else if result.RowsAffected() <= 0 {
		return nil, nil, pkgerrors.Wrapf(ErrNotExists, "app with ID %d does not exist", app.ID)
	}
	addedDaemons, deletedDaemons, err = updateAppDaemons(tx, app)
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem updating daemons for app %d", app.ID)
		return
	}
	// Update access points.
	err = updateAppAccessPoints(tx, app, true)
	if err != nil {
		err = pkgerrors.WithMessagef(err, "problem updating access points to app: %+v", app)
	}
	return
}

// Updates application in the database. It begins a new transaction when
// dbi has a *pg.DB type or uses an existing transaction when dbi has
// a *pg.Tx type.
func UpdateApp(dbi dbops.DBI, app *App) (addedDaemons []*Daemon, deletedDaemons []*Daemon, err error) {
	if db, ok := dbi.(*pg.DB); ok {
		err = db.RunInTransaction(context.Background(), func(tx *pg.Tx) error {
			addedDaemons, deletedDaemons, err = updateApp(tx, app)
			return err
		})
		return
	}
	addedDaemons, deletedDaemons, err = updateApp(dbi.(*pg.Tx), app)
	return
}

// Updates specified app's name. It returns app instance with old name and possibly
// an error. If an error occurs, a nil app is returned.
func RenameApp(dbi dbops.DBI, id int64, newName string) (*App, error) {
	app := &App{
		ID:   id,
		Name: newName,
	}
	// We're going to use the following query:
	// WITH rename AS (
	//     UPDATE app SET name = ?
	//     WHERE id = ?
	//     RETURNING name
	// )
	// SELECT * FROM app WHERE id = ?;

	// This query selects app name before doing and update and performs
	// the update. The idea for this query was taken from the PostgreSQL
	// documentation: https://www.postgresql.org/docs/9.1/queries-with.html

	updateQuery := dbi.Model(app).
		Column("name").
		WherePK().
		Returning("name")

	err := dbi.Model(app).
		WithUpdate("rename", updateQuery).
		WherePK().
		Select()
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem renaming app %d to %s", app.ID, newName)
	}

	return app, nil
}

// Returns an application with a given ID.
func GetAppByID(dbi dbops.DBI, id int64) (*App, error) {
	app := App{}
	q := dbi.Model(&app)
	q = q.Relation("Machine")
	q = q.Relation("AccessPoints")
	q = q.Relation("Daemons.KeaDaemon.KeaDHCPDaemon")
	q = q.Relation("Daemons.Bind9Daemon")
	q = q.Relation("Daemons.LogTargets")
	q = q.Where("app.id = ?", id)
	err := q.Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem getting app %v", id)
	}
	return &app, nil
}

// Returns applications belonging to a machine with a given ID.
func GetAppsByMachine(dbi dbops.DBI, machineID int64) ([]*App, error) {
	var apps []*App

	q := dbi.Model(&apps)
	q = q.Relation("AccessPoints")
	q = q.Relation("Daemons.KeaDaemon.KeaDHCPDaemon")
	q = q.Relation("Daemons.Bind9Daemon")
	q = q.Relation("Daemons.LogTargets")
	q = q.Relation("Daemons.ConfigReview")
	q = q.Where("machine_id = ?", machineID)
	q = q.OrderExpr("id ASC")
	err := q.Select()
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem getting apps")
	}
	return apps, nil
}

// Fetches all apps by type including the corresponding services.
func GetAppsByType(dbi dbops.DBI, appType AppType) ([]App, error) {
	var apps []App

	q := dbi.Model(&apps)
	q = q.Where("type = ?", appType)
	q = q.Relation("Machine")
	q = q.Relation("AccessPoints")
	q = q.Relation("Daemons.LogTargets")

	switch appType {
	case AppTypeKea:
		q = q.Relation("Daemons.Services.HAService")
		q = q.Relation("Daemons.KeaDaemon.KeaDHCPDaemon")
	case AppTypeBind9:
		q = q.Relation("Daemons.Bind9Daemon")
	}

	q = q.OrderExpr("id ASC")
	err := q.Select()
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem getting %s apps from database", appType)
	}
	return apps, nil
}

// Fetches a collection of apps from the database. The offset and
// limit specify the beginning of the page and the maximum size of the
// page. Limit has to be greater then 0, otherwise error is
// returned. sortField allows indicating sort column in database and
// sortDir allows selection the order of sorting. If sortField is
// empty then id is used for sorting. If SortDirAny is used then ASC
// order is used.
func GetAppsByPage(dbi dbops.DBI, offset int64, limit int64, filterText *string, appType AppType, sortField string, sortDir SortDirEnum) ([]App, int64, error) {
	if limit == 0 {
		return nil, 0, pkgerrors.New("limit should be greater than 0")
	}
	var apps []App

	// prepare query
	q := dbi.Model(&apps)
	q = q.Relation("AccessPoints")
	q = q.Relation("Machine")
	q = q.Relation("Daemons.KeaDaemon.KeaDHCPDaemon")
	q = q.Relation("Daemons.Bind9Daemon")
	q = q.Relation("Daemons.LogTargets")
	if appType != "" {
		q = q.Where("type = ?", appType)
	}
	if filterText != nil {
		text := "%" + *filterText + "%"
		q = q.WhereGroup(func(qq *orm.Query) (*orm.Query, error) {
			qq = qq.WhereOr("type ILIKE ?", text)
			qq = qq.WhereOr("name ILIKE ?", text)
			qq = qq.WhereOr("meta->>'Version' ILIKE ?", text)
			qq = qq.WhereOr("machine.address ILIKE ?", text)
			qq = qq.WhereOr("machine.state->>'Hostname' ILIKE ?", text)
			return qq, nil
		})
	}

	// prepare sorting expression, offset and limit
	ordExpr := prepareOrderExpr("app", sortField, sortDir)
	q = q.OrderExpr(ordExpr)
	q = q.Offset(int(offset))
	q = q.Limit(int(limit))

	total, err := q.SelectAndCount()
	if err != nil {
		if errors.Is(err, pg.ErrNoRows) {
			return []App{}, 0, nil
		}
		return nil, 0, pkgerrors.Wrapf(err, "problem getting apps")
	}
	return apps, int64(total), nil
}

// Retrieves all apps from the database. The second argument specifies if
// the app information must be returned with relations, i.e. with daemons,
// access points and machines information. If it is set to false, only
// the data belonging to the app table are returned.
func GetAllApps(dbi dbops.DBI, withRelations bool) ([]App, error) {
	return GetAllAppsWithRelations(dbi, AppRelationAccessPoints, AppRelationKeaDHCPDaemons, AppRelationBind9Daemons, AppRelationDaemonsLogTargets, AppRelationMachine)
}

// Retrieves all apps with custom relations. If no relations are specified
// it returns only the data contained in the app table.
func GetAllAppsWithRelations(dbi dbops.DBI, relations ...AppRelation) ([]App, error) {
	var apps []App
	q := dbi.Model(&apps)

	for _, relation := range relations {
		q = q.Relation(string(relation))
	}
	q = q.OrderExpr("id ASC")

	err := q.Select()
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem getting apps from the database")
	}
	return apps, nil
}

// Deletes an application from the database. Returns an error if the application
// doesn't exist.
func DeleteApp(dbi dbops.DBI, app *App) error {
	result, err := dbi.Model(app).WherePK().Delete()
	if err != nil {
		return pkgerrors.Wrapf(err, "problem deleting app %v", app.ID)
	} else if result.RowsAffected() <= 0 {
		return pkgerrors.Wrapf(ErrNotExists, "app with ID %d does not exist", app.ID)
	}
	return nil
}

// Returns a list of names of active DHCP daemons. This is useful for
// creating commands to be send to active DHCP servers.
func (app *App) GetActiveDHCPDaemonNames() (daemons []string) {
	if app.Type != AppTypeKea {
		return
	}
	for _, d := range app.Daemons {
		if d.Active && (d.Name == DaemonNameDHCPv4 || d.Name == DaemonNameDHCPv6) {
			daemons = append(daemons, d.Name)
		}
	}
	return
}

// Finds daemon by name. If a daemon with this name doesn't exist,
// a nil pointer is returned.
func (app *App) GetDaemonByName(name string) *Daemon {
	for _, daemon := range app.Daemons {
		if daemon.Name == name {
			return daemon
		}
	}
	return nil
}

// GetAccessPoint returns the access point of the given app and given access
// point type.
func (app *App) GetAccessPoint(accessPointType string) (ap *AccessPoint, err error) {
	for _, point := range app.AccessPoints {
		if point.Type == accessPointType {
			return point, nil
		}
	}
	return nil, pkgerrors.Errorf("no access point of type %s found for app ID %d", accessPointType, app.ID)
}

// AppTag implementation.

// Returns app ID.
func (app App) GetID() int64 {
	return app.ID
}

// Returns app name.
func (app App) GetName() string {
	return app.Name
}

// Returns app type.
func (app App) GetType() AppType {
	return app.Type
}

// Returns app version.
func (app App) GetVersion() string {
	return app.Meta.Version
}

// Return ID of a machine owning the app.
func (app App) GetMachineID() int64 {
	return app.MachineID
}

// Remaining functions for the agentcomm.ControlledApp implementation.

// Returns app control access point including control address, port and
// the flag indicating if the connection is secure.
func (app App) GetControlAccessPoint() (address string, port int64, key string, secure bool, err error) {
	var ap *AccessPoint
	ap, err = app.GetAccessPoint(AccessPointControl)
	if err == nil {
		address = ap.Address
		port = ap.Port
		key = ap.Key
		secure = ap.UseSecureProtocol
	}
	return
}

// Returns MachineTag interface to the machine owning the app.
func (app App) GetMachineTag() MachineTag {
	return app.Machine
}

// Returns DaemonTag interfaces to the daemons owned by the app.
func (app App) GetDaemonTags() (tags []DaemonTag) {
	for i := range app.Daemons {
		daemon := *app.Daemons[i]
		daemon.App = &app
		tags = append(tags, daemon)
	}
	return
}

// Returns selected daemon's tag or nil if the daemon does
// not exist.
func (app App) GetDaemonTag(daemonName string) DaemonTag {
	for i := range app.Daemons {
		if app.Daemons[i].Name == daemonName {
			return app.Daemons[i]
		}
	}
	return nil
}

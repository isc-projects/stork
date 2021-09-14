package dbmodel

import (
	"errors"
	"time"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	pkgerrors "github.com/pkg/errors"
	dbops "isc.org/stork/server/database"
)

const (
	AppTypeKea   = "kea"
	AppTypeBind9 = "bind9"
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
	Machine   *Machine
	Type      string // currently supported types are: "kea" and "bind9"
	Active    bool
	Meta      AppMeta
	Name      string

	AccessPoints []*AccessPoint

	Daemons []*Daemon
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
				return pkgerrors.Wrapf(err, "problem with removing access points from app %d", app.ID)
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
			return pkgerrors.Wrapf(err, "problem with adding access point to app %d: %v", app.ID, point)
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
		return nil, nil, pkgerrors.Wrapf(err, "problem with deleting daemons for an updated app %d", app.ID)
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
		} else {
			_, err = tx.Model(daemon).WherePK().Update()
		}
		if err != nil {
			return nil, nil, pkgerrors.Wrapf(err, "problem with upserting daemon to app %d: %v", app.ID, daemon)
		}

		if daemon.KeaDaemon != nil {
			// Make sure that the kea_daemon references the daemon.
			daemon.KeaDaemon.DaemonID = daemon.ID
			err = upsertInTransaction(tx, daemon.KeaDaemon.ID, daemon.KeaDaemon)
			if err != nil {
				return nil, nil, pkgerrors.Wrapf(err, "problem with upserting Kea daemon to app %d: %v",
					app.ID, daemon.KeaDaemon)
			}

			if daemon.KeaDaemon.KeaDHCPDaemon != nil {
				// Make sure that the kea_dhcp_daemon references the kea_daemon.
				daemon.KeaDaemon.KeaDHCPDaemon.KeaDaemonID = daemon.KeaDaemon.ID
				err = upsertInTransaction(tx, daemon.KeaDaemon.KeaDHCPDaemon.ID, daemon.KeaDaemon.KeaDHCPDaemon)
				if err != nil {
					return nil, nil, pkgerrors.Wrapf(err, "problem with upserting Kea DHCP daemon to app %d: %v",
						app.ID, daemon.KeaDaemon.KeaDHCPDaemon)
				}
			}
		} else if daemon.Bind9Daemon != nil {
			// Make sure that the bind9_daemon references the daemon.
			daemon.Bind9Daemon.DaemonID = daemon.ID
			if daemon.Bind9Daemon.ID == 0 {
				_, err = tx.Model(daemon.Bind9Daemon).Insert()
			} else {
				_, err = tx.Model(daemon.Bind9Daemon).WherePK().Update()
			}
			if err != nil {
				return nil, nil, pkgerrors.Wrapf(err, "problem with upserting BIND9 daemon to app %d: %v",
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
			return nil, nil, pkgerrors.Wrapf(err, "problem with deleting log targets for updated daemon %d",
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
			} else {
				_, err = tx.Model(daemon.LogTargets[i]).WherePK().Update()
			}
			if err != nil {
				return nil, nil, pkgerrors.Wrapf(err, "problem with upserting log target %s to daemon %d: %v",
					daemon.LogTargets[i].Output, daemon.ID, daemon)
			}
		}
	}
	return addedDaemons, deletedDaemons, nil
}

// Adds application into the database. The dbIface object may either
// be a pg.DB object or pg.Tx. In the latter case this function uses
// existing transaction to add an app. Returns a list of added daemons
// and error if occurred.
func AddApp(dbIface interface{}, app *App) ([]*Daemon, error) {
	// Start transaction if it hasn't been started yet.
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		return nil, err
	}
	// Always rollback when this function ends. If the changes get committed
	// first this is no-op.
	defer rollback()

	err = tx.Insert(app)
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem with inserting app %v", app)
	}

	addedDaemons, deletedDaemons, err := updateAppDaemons(tx, app)
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem with inserting daemons for a new app")
	}
	if len(deletedDaemons) > 0 {
		return nil, pkgerrors.Errorf("problem with deleting daemons for a new app")
	}

	// Add access points.
	err = updateAppAccessPoints(tx, app, false)
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem with adding access points to app: %+v", app)
	}

	// Commit the changes if necessary.
	err = commit()
	if err != nil {
		return nil, err
	}

	return addedDaemons, nil
}

// Updates application in the database. An error is returned if the
// app does not exist. The dbIface object may either be a pg.DB object
// or pg.Tx. In the latter case this function uses existing
// transaction to add an app. Returns a list of added daemons, deleted
// daemons and error if occurred.
func UpdateApp(dbIface interface{}, app *App) ([]*Daemon, []*Daemon, error) {
	// Start transaction if it hasn't been started yet.
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		return nil, nil, err
	}
	// Always rollback when this function ends. If the changes get committed
	// first this is no-op.
	defer rollback()

	err = tx.Update(app)
	if err != nil {
		return nil, nil, pkgerrors.Wrapf(err, "problem with updating app %v", app)
	}

	addedDaemons, deletedDaemons, err := updateAppDaemons(tx, app)
	if err != nil {
		return nil, nil, pkgerrors.WithMessagef(err, "problem with updating daemons for app %d", app.ID)
	}

	// Update access points.
	err = updateAppAccessPoints(tx, app, true)
	if err != nil {
		return nil, nil, pkgerrors.WithMessagef(err, "problem with updating access points to app: %+v", app)
	}

	// Commit the changes if necessary.
	err = commit()
	if err != nil {
		return nil, nil, err
	}

	return addedDaemons, deletedDaemons, nil
}

// Insert or update the application in the database.
// First, the function tries to insert a new entry
// into the database. If it volatiles the unique constraint
// (entry is already added), then the update query is performed.
// Returns a list of added daemons, deleted daemons (only if the
// update performs), a flag that indicates that the INSERT query
// executed and error if occurred.
// This function is composed from 3 stages.
// 1. Insert app
//    It is executed when AppID equals 0. It
//    If any other query inserted the app before this query then the transaction rollbacks
//    else the function successfully returns.
// 2. Retrieve app ID.
//    If the app exists then we need an application ID.
// 3. Update the application.
func AddOrUpdateApp(db *pg.DB, app *App) ([]*Daemon, []*Daemon, bool, error) {
	hasInsertConflict := false
	if app.ID == 0 {
		// New app, insert it.
		addedDaemons, err := AddApp(db, app)

		if err == nil {
			return addedDaemons, nil, true, nil
		}

		// AddApp fails
		var pgErr pg.Error
		ok := errors.As(pkgerrors.Cause(err), &pgErr)
		// Is it a conflict on an unique constraint?
		if !ok || !pgErr.IntegrityViolation() {
			return nil, nil, false, err
		}

		hasInsertConflict = true
	}

	// Stage 2: Check conflict and retrieve AppID if needed
	if hasInsertConflict {
		// Get App ID
		existingApp, err := getAppByName(db, app.Name)
		if err != nil {
			return nil, nil, false, err
		}
		app.ID = existingApp.ID
		app.CreatedAt = existingApp.CreatedAt
	}

	// Stage 3: Update application
	// Existing app, update it if needed.
	addedDaemons, deletedDaemons, err := UpdateApp(db, app)
	if err != nil {
		err = pkgerrors.Wrapf(err, "Invalid update for app: %+v, hasInsertConflict: %t", app, hasInsertConflict)
	}
	return addedDaemons, deletedDaemons, false, err
}

// Updates specified app's name. It returns app instance with old name and possibly
// an error. If an error occurs, a nil app is returned.
func RenameApp(db *pg.DB, id int64, newName string) (*App, error) {
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

	updateQuery := db.Model(app).
		Column("name").
		WherePK().
		Returning("name")

	err := db.Model(app).
		WithUpdate("rename", updateQuery).
		WherePK().
		Select()
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem with renaming an app %d to %s", app.ID, newName)
	}

	return app, nil
}

func GetAppByID(db *pg.DB, id int64) (*App, error) {
	app := App{}
	q := db.Model(&app)
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
		return nil, pkgerrors.Wrapf(err, "problem with getting app %v", id)
	}
	return &app, nil
}

// Return app with given name as the name has a unique constraint.
// It doesn't include related entries. Intended to the internal use.
func getAppByName(db *pg.DB, name string) (*App, error) {
	app := App{}
	q := db.Model(&app)
	q = q.Where("app.name = ?", name)
	err := q.Select()
	return &app, err
}

func GetAppsByMachine(db *pg.DB, machineID int64) ([]*App, error) {
	var apps []*App

	q := db.Model(&apps)
	q = q.Relation("AccessPoints")
	q = q.Relation("Daemons.KeaDaemon.KeaDHCPDaemon")
	q = q.Relation("Daemons.Bind9Daemon")
	q = q.Relation("Daemons.LogTargets")
	q = q.Where("machine_id = ?", machineID)
	q = q.OrderExpr("id ASC")
	err := q.Select()
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem with getting apps")
	}
	return apps, nil
}

// Fetches all apps by type including the corresponding services.
func GetAppsByType(db *pg.DB, appType string) ([]App, error) {
	var apps []App

	q := db.Model(&apps)
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
		return nil, pkgerrors.Wrapf(err, "problem with getting %s apps from database", appType)
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
func GetAppsByPage(db *pg.DB, offset int64, limit int64, filterText *string, appType string, sortField string, sortDir SortDirEnum) ([]App, int64, error) {
	if limit == 0 {
		return nil, 0, pkgerrors.New("limit should be greater than 0")
	}
	var apps []App

	// prepare query
	q := db.Model(&apps)
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
		return nil, 0, pkgerrors.Wrapf(err, "problem with getting apps")
	}
	return apps, int64(total), nil
}

// Retrieves all apps from the database. The second argument specifies if
// the app information must be returned with relations, i.e. with daemons,
// access points and machines information. If it is set to false, only
// the data belonging to the app table are returned.
func GetAllApps(db *pg.DB, withRelations bool) ([]App, error) {
	var apps []App

	// prepare query
	q := db.Model(&apps)

	if withRelations {
		q = q.Relation("AccessPoints")
		q = q.Relation("Daemons.KeaDaemon.KeaDHCPDaemon")
		q = q.Relation("Daemons.Bind9Daemon")
		q = q.Relation("Daemons.LogTargets")
		q = q.Relation("Machine")
	}
	q = q.OrderExpr("id ASC")

	// retrieve apps from db
	err := q.Select()
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem with getting apps")
	}
	return apps, nil
}

func DeleteApp(db *pg.DB, app *App) error {
	err := db.Delete(app)
	if err != nil {
		return pkgerrors.Wrapf(err, "problem with deleting app %v", app.ID)
	}
	return nil
}

// Returns a list of names of active DHCP daemons. This is useful for
// creating commands to be send to active DHCP servers.
func (app *App) GetActiveDHCPDaemonNames() (daemons []string) {
	if app.Type != AppTypeKea {
		return daemons
	}
	for _, d := range app.Daemons {
		if d.Active && (d.Name == DaemonNameDHCPv4 || d.Name == DaemonNameDHCPv6) {
			daemons = append(daemons, d.Name)
		}
	}
	return daemons
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

// Returns local subnet ID for a given subnet prefix. If subnets have been indexed,
// this function will use the index to find a subnet with the matching prefix. This
// is much faster, but requires that the caller first builds a collection of
// indexed subnets and associates it with appropriate KeaDHCPDaemon instances.
// If the indexes are not built, this function will simply iterate over the
// subnets within the configuration. This is generally much slower and should
// be only used for sporadic calls to GetLocalSubnetID().
// If the matching subnet is not found, the 0 value is returned.
func (app *App) GetLocalSubnetID(prefix string) int64 {
	if app.Type != AppTypeKea {
		return 0
	}
	for _, d := range app.Daemons {
		if d.KeaDaemon == nil {
			continue
		}
		// If subnets happen to be indexed, use the index to locate the subnet because
		// it is a lot faster than full scan.
		if d.KeaDaemon.KeaDHCPDaemon != nil && d.KeaDaemon.KeaDHCPDaemon.IndexedSubnets != nil {
			is := d.KeaDaemon.KeaDHCPDaemon.IndexedSubnets
			if subnet, ok := is.ByPrefix[prefix]; ok {
				if id, ok := subnet["id"].(float64); ok {
					return int64(id)
				}
			}

			continue
		}

		// No index for subnets for this daemon. We have to do full subnets scan.
		if d.KeaDaemon.Config != nil {
			if id := d.KeaDaemon.Config.GetLocalSubnetID(prefix); id > 0 {
				return id
			}
		}
	}
	return 0
}

// GetAccessPoint returns the access point of the given app and given access
// point type.
func (app *App) GetAccessPoint(accessPointType string) (ap *AccessPoint, err error) {
	for _, point := range app.AccessPoints {
		if point.Type == accessPointType {
			return point, nil
		}
	}
	return nil, pkgerrors.Errorf("no access point of type %s found for app id %d", accessPointType, app.ID)
}

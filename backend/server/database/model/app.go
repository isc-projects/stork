package dbmodel

import (
	"time"

	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	"github.com/pkg/errors"

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
		q := tx.Model((*AccessPoint)(nil))
		q = q.Where("app_id = ?", app.ID)
		q = q.Where("type NOT IN (?)", pg.In(types))
		_, err = q.Delete()
		if err != nil {
			return errors.Wrapf(err, "problem with removing access points from app %d", app.ID)
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
			return errors.Wrapf(err, "problem with adding access point to app %d: %v", app.ID, point)
		}
	}
	return nil
}

// Adds or updates daemons for an app. If any daemons already exist for the app,
// they are removed and the new daemons will be added instead.
func updateAppDaemons(tx *pg.Tx, app *App) (err error) {
	// Delete the existing daemons, because the updated app may have different
	// set of daemons. In particular, some of them may be gone.
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
	_, err = q.Delete()
	if err != nil {
		return errors.Wrapf(err, "problem with deleting daemons for an updated app %d", app.ID)
	}

	// Add updated daemons.
	for i := range app.Daemons {
		daemon := app.Daemons[i]

		// Make sure the inserted daemon references the app.
		daemon.AppID = app.ID
		if daemon.ID == 0 {
			// Add the new entry to the daemon table.
			_, err = tx.Model(daemon).Insert()
		} else {
			_, err = tx.Model(daemon).WherePK().Update()
		}
		if err != nil {
			return errors.Wrapf(err, "problem with upserting daemon to app %d: %v", app.ID, daemon)
		}

		if daemon.KeaDaemon != nil {
			// Make sure that the kea_daemon references the daemon.
			daemon.KeaDaemon.DaemonID = daemon.ID
			if daemon.KeaDaemon.ID == 0 {
				_, err = tx.Model(daemon.KeaDaemon).Insert()
			} else {
				_, err = tx.Model(daemon.KeaDaemon).WherePK().Update()
			}
			if err != nil {
				return errors.Wrapf(err, "problem with upserting Kea daemon to app %d: %v",
					app.ID, daemon.KeaDaemon)
			}

			if daemon.KeaDaemon.KeaDHCPDaemon != nil {
				// Make sure that the kea_dhcp_daemon references the kea_daemon.
				daemon.KeaDaemon.KeaDHCPDaemon.KeaDaemonID = daemon.KeaDaemon.ID
				if daemon.KeaDaemon.KeaDHCPDaemon.ID == 0 {
					_, err = tx.Model(daemon.KeaDaemon.KeaDHCPDaemon).Insert()
				} else {
					_, err = tx.Model(daemon.KeaDaemon.KeaDHCPDaemon).WherePK().Update()
				}
				if err != nil {
					return errors.Wrapf(err, "problem with upserting Kea DHCP daemon to app %d: %v",
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
				return errors.Wrapf(err, "problem with upserting BIND9 daemon to app %d: %v",
					app.ID, daemon.Bind9Daemon)
			}
		}
	}
	return nil
}

// Adds application into the database. The dbIface object may either be a pg.DB
// object or pg.Tx. In the latter case this function uses existing transaction
// to add an app.
func AddApp(dbIface interface{}, app *App) error {
	// Start transaction if it hasn't been started yet.
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		return err
	}
	// Always rollback when this function ends. If the changes get committed
	// first this is no-op.
	defer rollback()

	err = tx.Insert(app)
	if err != nil {
		return errors.Wrapf(err, "problem with inserting app %v", app)
	}

	err = updateAppDaemons(tx, app)
	if err != nil {
		return errors.WithMessagef(err, "problem with inserting daemons for a new app")
	}

	// Add access points.
	err = updateAppAccessPoints(tx, app, false)
	if err != nil {
		return errors.Wrapf(err, "problem with adding access points to app: %+v", app)
	}

	// Commit the changes if necessary.
	err = commit()
	return err
}

// Updates application in the database. An error is returned if the app
// does not exist. The dbIface object may either be a pg.DB object or
// pg.Tx. In the latter case this function uses existing transaction
// to add an app.
func UpdateApp(dbIface interface{}, app *App) error {
	// Start transaction if it hasn't been started yet.
	tx, rollback, commit, err := dbops.Transaction(dbIface)
	if err != nil {
		return err
	}
	// Always rollback when this function ends. If the changes get committed
	// first this is no-op.
	defer rollback()

	err = tx.Update(app)
	if err != nil {
		return errors.Wrapf(err, "problem with updating app %v", app)
	}

	err = updateAppDaemons(tx, app)
	if err != nil {
		return errors.WithMessagef(err, "problem with updating daemons for app %d", app.ID)
	}

	// Update access points.
	err = updateAppAccessPoints(tx, app, true)
	if err != nil {
		return errors.Wrapf(err, "problem with updating access points to app: %+v", app)
	}

	// Commit the changes if necessary.
	err = commit()
	return err
}

func GetAppByID(db *pg.DB, id int64) (*App, error) {
	app := App{}
	q := db.Model(&app)
	q = q.Relation("Machine")
	q = q.Relation("AccessPoints")
	q = q.Relation("Daemons.KeaDaemon.KeaDHCPDaemon")
	q = q.Relation("Daemons.Bind9Daemon")
	q = q.Where("app.id = ?", id)
	err := q.Select()
	if err == pg.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem with getting app %v", id)
	}
	return &app, nil
}

func GetAppsByMachine(db *pg.DB, machineID int64) ([]App, error) {
	var apps []App

	q := db.Model(&apps)
	q = q.Relation("AccessPoints")
	q = q.Relation("Daemons.KeaDaemon.KeaDHCPDaemon")
	q = q.Relation("Daemons.Bind9Daemon")
	q = q.Where("machine_id = ?", machineID)
	q = q.OrderExpr("id ASC")
	err := q.Select()
	if err != nil {
		return nil, errors.Wrapf(err, "problem with getting apps")
	}
	return apps, nil
}

// Fetches all app by type from the database.
func GetAppsByType(db *pg.DB, appType string) ([]App, error) {
	var apps []App

	q := db.Model(&apps)
	q = q.Where("type = ?", appType)
	q = q.Relation("Machine")
	q = q.Relation("AccessPoints")

	switch appType {
	case AppTypeKea:
		q = q.Relation("Daemons.KeaDaemon.KeaDHCPDaemon")
	case AppTypeBind9:
		q = q.Relation("Daemons.Bind9Daemon")
	}

	q = q.OrderExpr("id ASC")
	err := q.Select()
	if err != nil {
		return nil, errors.Wrapf(err, "problem with getting %s apps", appType)
	}
	return apps, nil
}

// Fetches a collection of apps from the database. The offset and limit specify the
// beginning of the page and the maximum size of the page. Limit has to be greater
// then 0, otherwise error is returned.
func GetAppsByPage(db *pg.DB, offset int64, limit int64, text string, appType string) ([]App, int64, error) {
	if limit == 0 {
		return nil, 0, errors.New("limit should be greater than 0")
	}
	var apps []App

	// prepare query
	q := db.Model(&apps)
	q = q.Relation("AccessPoints")
	q = q.Relation("Machine")
	q = q.Relation("Daemons.KeaDaemon.KeaDHCPDaemon")
	q = q.Relation("Daemons.Bind9Daemon")
	if appType != "" {
		q = q.Where("type = ?", appType)
	}
	if text != "" {
		text = "%" + text + "%"
		q = q.WhereGroup(func(qq *orm.Query) (*orm.Query, error) {
			qq = qq.WhereOr("meta->>'Version' ILIKE ?", text)
			return qq, nil
		})
	}

	// and then, first get total count
	total, err := q.Clone().Count()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "problem with getting apps total")
	}

	// then retrieve given page of rows
	q = q.Order("id ASC").Offset(int(offset)).Limit(int(limit))
	err = q.Select()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "problem with getting apps")
	}
	return apps, int64(total), nil
}

func GetAllApps(db *pg.DB) ([]App, error) {
	var apps []App

	// prepare query
	q := db.Model(&apps)
	q = q.Relation("AccessPoints")
	q = q.Relation("Daemons.KeaDaemon.KeaDHCPDaemon")
	q = q.Relation("Daemons.Bind9Daemon")
	q = q.OrderExpr("id ASC")

	// retrieve apps from db
	err := q.Select()
	if err != nil {
		return nil, errors.Wrapf(err, "problem with getting apps")
	}
	return apps, nil
}

func DeleteApp(db *pg.DB, app *App) error {
	err := db.Delete(app)
	if err != nil {
		return errors.Wrapf(err, "problem with deleting app %v", app.ID)
	}
	return nil
}

// Returns a list of names of active DHCP deamons. This is useful for
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

// Returns local subnet ID for a given subnet prefix. It iterates over the
// daemons' configurations and searches for a subnet with matching prefix.
// If the match isn't found, the value of 0 is returned.
func (app *App) GetLocalSubnetID(prefix string) int64 {
	if app.Type != AppTypeKea {
		return 0
	}
	for _, d := range app.Daemons {
		if d.KeaDaemon == nil || d.KeaDaemon.Config == nil {
			continue
		}
		if id := d.KeaDaemon.Config.GetLocalSubnetID(prefix); id > 0 {
			return id
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
	return nil, errors.Errorf("no access point of type %s found for app id %d", accessPointType, app.ID)
}

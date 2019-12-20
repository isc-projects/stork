package dbmodel

import (
	"time"
	"encoding/json"
	"github.com/go-pg/pg/v9"
	"github.com/go-pg/pg/v9/orm"
	"github.com/pkg/errors"
	"isc.org/stork"
)

type KeaDaemon struct {
	Pid int32
	Name string
	Active bool
	Version string
	ExtendedVersion string
}

type AppKea struct {
	ExtendedVersion  string
	Daemons          []KeaDaemon
}

type AppBind struct {
}

// Part of app table in database that describes metadata of app. In DB it is stored as JSONB.
type AppMeta struct {
	Version          string
}

// Represents a app held in app table in the database.
type App struct {
	Id           int64
	Created      time.Time
	Deleted      time.Time
	MachineID    int64
	Machine      Machine
	Type         string
	CtrlPort     int64
	Active       bool
	Meta         AppMeta
	Details      interface{}
}

func AddApp(db *pg.DB, app *App) error {
	err := db.Insert(app)
	if err != nil {
		return errors.Wrapf(err, "problem with inserting app %v", app)
	}
	return nil
}

func ReconvertAppDetails(app *App) error {
	bytes, err := json.Marshal(app.Details)
	if err != nil {
		return errors.Wrapf(err, "problem with getting app from db: %v ", app)
	}
	var s AppKea
	err = json.Unmarshal(bytes, &s)
	if err != nil {
		return errors.Wrapf(err, "problem with getting app from db: %v ", app)
	}
	app.Details = s
	return nil
}

func GetAppById(db *pg.DB, id int64) (*App, error) {
	app := App{}
	q := db.Model(&app).Where("app.id = ?", id)
	q = q.Relation("Machine")
	err := q.Select()
	if err == pg.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem with getting app %v", id)
	}
	err = ReconvertAppDetails(&app)
	if err != nil {
		return nil, err
	}
	return &app, nil
}

func GetAppsByMachine(db *pg.DB, machineId int64) ([]App, error) {
	var apps []App

	q := db.Model(&apps)
	q = q.Where("machine_id = ?", machineId)
	err := q.Select()
	if err != nil {
		return nil, errors.Wrapf(err, "problem with getting apps")
	}

	for idx := range apps {
		err = ReconvertAppDetails(&apps[idx])
		if err != nil {
			return nil, err
		}
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
	q = q.Where("app.deleted is NULL")
	q = q.Relation("Machine")
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

	// then retrive given page of rows
	q = q.Order("id ASC").Offset(int(offset)).Limit(int(limit))
	err = q.Select()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "problem with getting apps")
	}
	for idx := range apps {
		err = ReconvertAppDetails(&apps[idx])
		if err != nil {
			return nil, 0, err
		}
	}
	return apps, int64(total), nil
}

func DeleteApp(db *pg.DB, app *App) error {
	app.Deleted = stork.UTCNow()
	err := db.Update(app)
	if err != nil {
		return errors.Wrapf(err, "problem with deleting app %v", app.Id)
	}
	return nil
}

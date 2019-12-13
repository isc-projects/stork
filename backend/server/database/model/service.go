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

type ServiceKea struct {
	ExtendedVersion  string
	Daemons          []KeaDaemon
}

type ServiceBind struct {
}

// Part of service table in database that describes metadata of service. In DB it is stored as JSONB.
type ServiceMeta struct {
	Version          string
}

// Represents a service held in service table in the database.
type Service struct {
	Id           int64
	Created      time.Time
	Deleted      time.Time
	MachineID    int64
	Machine      Machine
	Type         string
	CtrlPort     int64
	Active       bool
	Meta         ServiceMeta
	Details      interface{}
}

func AddService(db *pg.DB, service *Service) error {
	err := db.Insert(service)
	if err != nil {
		return errors.Wrapf(err, "problem with inserting service %v", service)
	}
	return nil
}

func ReconvertServiceDetails(service *Service) error {
	bytes, err := json.Marshal(service.Details)
	if err != nil {
		return errors.Wrapf(err, "problem with getting service from db: %v ", service)
	}
	var s ServiceKea
	err = json.Unmarshal(bytes, &s)
	if err != nil {
		return errors.Wrapf(err, "problem with getting service from db: %v ", service)
	}
	service.Details = s
	return nil
}

func GetServiceById(db *pg.DB, id int64) (*Service, error) {
	service := Service{}
	q := db.Model(&service).Where("service.id = ?", id)
	q = q.Relation("Machine")
	err := q.Select()
	if err == pg.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrapf(err, "problem with getting service %v", id)
	}
	err = ReconvertServiceDetails(&service)
	if err != nil {
		return nil, err
	}
	return &service, nil
}

func GetServicesByMachine(db *pg.DB, machineId int64) ([]Service, error) {
	var services []Service

	q := db.Model(&services)
	q = q.Where("machine_id = ?", machineId)
	err := q.Select()
	if err != nil {
		return nil, errors.Wrapf(err, "problem with getting services")
	}

	for idx := range services {
		err = ReconvertServiceDetails(&services[idx])
		if err != nil {
			return nil, err
		}
	}
	return services, nil
}

// Fetches a collection of services from the database. The offset and limit specify the
// beginning of the page and the maximum size of the page. Limit has to be greater
// then 0, otherwise error is returned.
func GetServicesByPage(db *pg.DB, offset int64, limit int64, text string, serviceType string) ([]Service, int64, error) {
	if limit == 0 {
		return nil, 0, errors.New("limit should be greater than 0")
	}
	var services []Service

	// prepare query
	q := db.Model(&services)
	q = q.Where("service.deleted is NULL")
	q = q.Relation("Machine")
	if serviceType != "" {
		q = q.Where("type = ?", serviceType)
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
		return nil, 0, errors.Wrapf(err, "problem with getting services total")
	}

	// then retrive given page of rows
	q = q.Order("id ASC").Offset(int(offset)).Limit(int(limit))
	err = q.Select()
	if err != nil {
		return nil, 0, errors.Wrapf(err, "problem with getting services")
	}
	for idx := range services {
		err = ReconvertServiceDetails(&services[idx])
		if err != nil {
			return nil, 0, err
		}
	}
	return services, int64(total), nil
}

func DeleteService(db *pg.DB, service *Service) error {
	service.Deleted = stork.UTCNow()
	err := db.Update(service)
	if err != nil {
		return errors.Wrapf(err, "problem with deleting service %v", service.Id)
	}
	return nil
}

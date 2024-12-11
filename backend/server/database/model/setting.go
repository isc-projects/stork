package dbmodel

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/go-pg/pg/v10"
	pkgerrors "github.com/pkg/errors"
)

// This module provides global settings that can be used anywhere in the code.
// All settings with their default values are defined in defaultSettings table,
// in InitializeSettings function. There are a few functions for getting and
// setting these settings. Generally setters are used in API function
// so users can set these settings. Getters are used around the code
// where a given setting is needed.

// TODO: add caching to avoid trips to database; candidate caching libs:
// https://allegro.tech/2016/03/writing-fast-cache-service-in-go.html

// Valid data types for settings.
const (
	SettingValTypeInt    = 1
	SettingValTypeBool   = 2
	SettingValTypeStr    = 3
	SettingValTypePasswd = 4
)

// Represents a setting held in setting table in the database.
type Setting struct {
	Name    string `pg:",pk"`
	ValType int64
	Value   string `pg:",use_zero"`
}

// Initialize settings in db. If new setting needs to be added then add it to defaultSettings list
// and it will be automatically added to db here in this function.
// You can provide the interval used to initialize the database settings for
// pullers. Specify zero to use default values.
func InitializeSettings(db *pg.DB, initialPullerInterval int64) error {
	// Init puller intervals.
	longInterval := "60"
	mediumInterval := "30"

	if initialPullerInterval > 0 {
		interval := fmt.Sprint(initialPullerInterval)
		longInterval = interval
		mediumInterval = interval
	}

	// list of all stork settings with default values
	defaultSettings := []Setting{
		{
			Name:    "bind9_stats_puller_interval", // in seconds
			ValType: SettingValTypeInt,
			Value:   longInterval,
		},
		{
			Name:    "kea_stats_puller_interval", // in seconds
			ValType: SettingValTypeInt,
			Value:   longInterval,
		},
		{
			Name:    "kea_hosts_puller_interval", // in seconds
			ValType: SettingValTypeInt,
			Value:   longInterval,
		},
		{
			Name:    "kea_status_puller_interval", // in seconds
			ValType: SettingValTypeInt,
			Value:   mediumInterval,
		},
		{
			Name:    "apps_state_puller_interval", // in seconds
			ValType: SettingValTypeInt,
			Value:   mediumInterval,
		},
		{
			Name:    "grafana_url",
			ValType: SettingValTypeStr,
			Value:   "",
		},
		{
			Name:    "enable_machine_registration",
			ValType: SettingValTypeBool,
			Value:   "true",
		},
		{
			Name:    "enable_online_software_versions",
			ValType: SettingValTypeBool,
			Value:   "true",
		},
	}

	// Check if there are new settings vs existing ones. Add new ones to DB.
	_, err := db.Model(&defaultSettings).OnConflict("DO NOTHING").Insert()
	if err != nil {
		err = pkgerrors.Wrapf(err, "problem inserting default settings")
	}
	return err
}

// Get setting record from db based on its name.
func GetSetting(db *pg.DB, name string) (*Setting, error) {
	setting := Setting{}
	q := db.Model(&setting).Where("setting.name = ?", name)
	err := q.Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, pkgerrors.Wrapf(err, "setting %s is missing", name)
	} else if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem getting setting %s", name)
	}
	return &setting, nil
}

// Get setting by name and check if its type matches to expected one.
func getAndCheckSetting(db *pg.DB, name string, expValType int64) (*Setting, error) {
	s, err := GetSetting(db, name)
	if err != nil {
		return nil, err
	}
	if s.ValType != expValType {
		return nil, pkgerrors.Errorf("no matching setting type of %s (%d vs %d expected)", name, s.ValType, expValType)
	}
	return s, nil
}

// Get int value of given setting by name.
func GetSettingInt(db *pg.DB, name string) (int64, error) {
	s, err := getAndCheckSetting(db, name, SettingValTypeInt)
	if err != nil {
		return 0, err
	}
	val, err := strconv.ParseInt(s.Value, 10, 64)
	if err != nil {
		return 0, err
	}
	return val, nil
}

// Get bool value of given setting by name.
func GetSettingBool(db *pg.DB, name string) (bool, error) {
	s, err := getAndCheckSetting(db, name, SettingValTypeBool)
	if err != nil {
		return false, err
	}
	val, err := strconv.ParseBool(s.Value)
	if err != nil {
		return false, err
	}
	return val, nil
}

// Get string value of given setting by name.
func GetSettingStr(db *pg.DB, name string) (string, error) {
	s, err := getAndCheckSetting(db, name, SettingValTypeStr)
	if err != nil {
		return "", err
	}
	return s.Value, nil
}

// Get password value of given setting by name.
func GetSettingPasswd(db *pg.DB, name string) (string, error) {
	s, err := getAndCheckSetting(db, name, SettingValTypePasswd)
	if err != nil {
		return "", err
	}
	return s.Value, nil
}

// Get all settings.
func GetAllSettings(db *pg.DB) (map[string]interface{}, error) {
	settings := []*Setting{}
	q := db.Model(&settings)
	err := q.Select()
	if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem getting all settings")
	}

	settingsMap := make(map[string]interface{})

	for _, s := range settings {
		switch s.ValType {
		case SettingValTypeInt:
			val, err := strconv.ParseInt(s.Value, 10, 64)
			if err != nil {
				return nil, pkgerrors.Wrapf(err, "problem getting setting value of %s", s.Name)
			}
			settingsMap[s.Name] = val
		case SettingValTypeBool:
			val, err := strconv.ParseBool(s.Value)
			if err != nil {
				return nil, pkgerrors.Wrapf(err, "problem getting setting value of %s", s.Name)
			}
			settingsMap[s.Name] = val
		case SettingValTypeStr:
			settingsMap[s.Name] = s.Value
		case SettingValTypePasswd:
			// do not return passwords to users
		}
	}

	return settingsMap, nil
}

// Set int value of given setting by name.
func SetSettingInt(db *pg.DB, name string, value int64) error {
	s, err := getAndCheckSetting(db, name, SettingValTypeInt)
	if err != nil {
		return err
	}
	s.Value = strconv.FormatInt(value, 10)
	result, err := db.Model(s).WherePK().Update()
	if err != nil {
		return pkgerrors.Wrapf(err, "problem updating setting %s", name)
	} else if result.RowsAffected() <= 0 {
		return pkgerrors.Wrapf(ErrNotExists, "configuration setting %s does not exist", name)
	}
	return nil
}

// Set bool value of given setting by name.
func SetSettingBool(db *pg.DB, name string, value bool) error {
	s, err := getAndCheckSetting(db, name, SettingValTypeBool)
	if err != nil {
		return err
	}
	s.Value = strconv.FormatBool(value)
	result, err := db.Model(s).WherePK().Update()
	if err != nil {
		return pkgerrors.Wrapf(err, "problem updating setting %s", name)
	} else if result.RowsAffected() <= 0 {
		return pkgerrors.Wrapf(ErrNotExists, "configuration setting %s does not exist", name)
	}
	return nil
}

// Set string value of given setting by name.
func SetSettingStr(db *pg.DB, name string, value string) error {
	s, err := getAndCheckSetting(db, name, SettingValTypeStr)
	if err != nil {
		return err
	}
	s.Value = value
	result, err := db.Model(s).WherePK().Update()
	if err != nil {
		return pkgerrors.Wrapf(err, "problem updating setting %s", name)
	} else if result.RowsAffected() <= 0 {
		return pkgerrors.Wrapf(ErrNotExists, "configuration setting %s does not exist", name)
	}
	return nil
}

// Set password value of given setting by name.
func SetSettingPasswd(db *pg.DB, name string, value string) error {
	s, err := getAndCheckSetting(db, name, SettingValTypePasswd)
	if err != nil {
		return err
	}
	s.Value = value
	result, err := db.Model(s).WherePK().Update()
	if err != nil {
		return pkgerrors.Wrapf(err, "problem updating setting %s", name)
	} else if result.RowsAffected() <= 0 {
		return pkgerrors.Wrapf(ErrNotExists, "configuration setting %s does not exist", name)
	}
	return nil
}

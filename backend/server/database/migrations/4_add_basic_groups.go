package dbmigs

import (
	"github.com/go-pg/migrations/v7"
	"github.com/go-pg/pg/v9/orm"
)

func init() {
	type systemUser struct {
		Id int
	}

	type systemGroup struct {
		Id          int
		Name        string
		Description string
	}

	type systemUserToGroup struct {
		UserID int         `pg:",pk,not_null,on_delete:CASCADE"`
		User   *systemUser `json:"-"`

		GroupID int          `pg:",pk,not_null,on_delete:CASCADE"`
		Group   *systemGroup `json:"-"`
	}

	migrations.MustRegisterTx(func(db migrations.DB) error {
		for _, model := range []interface{}{&systemGroup{}, &systemUserToGroup{}} {
			err := db.Model(model).CreateTable(&orm.CreateTableOptions{
				FKConstraints: true,
				IfNotExists:   true,
			})
			if err != nil {
				return err
			}
		}

		adminGroup := &systemGroup{
			Name:        "super-admin",
			Description: "This group of users can access all system components.",
		}
		_, err := db.Model(adminGroup).Returning("id").Insert()
		if err != nil {
			return err
		}

		_, err = db.Model(&systemGroup{
			Name:        "admin",
			Description: "This group of users cannot manage user accounts.",
		}).Insert()
		if err != nil {
			return err
		}

		adminAssoc := systemUserToGroup{
			UserID:  1,
			GroupID: adminGroup.Id,
		}
		_, err = db.Model(&adminAssoc).Insert()
		if err != nil {
			return err
		}

		_, err = db.Exec(`WITH non_admin_assoc AS (
                            SELECT id, 2 FROM system_user WHERE id != 1
                          )
                          INSERT INTO system_user_to_group SELECT * FROM non_admin_assoc`)

		if err != nil {
			return err
		}

		return nil
	}, func(db migrations.DB) error {
		for _, model := range []interface{}{&systemGroup{}, &systemUserToGroup{}} {
			err := db.Model(model).DropTable(&orm.DropTableOptions{
				IfExists: true,
				Cascade:  true,
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
}

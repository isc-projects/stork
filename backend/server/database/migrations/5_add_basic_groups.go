package dbmigs

import (
	"github.com/go-pg/migrations/v8"
	"github.com/go-pg/pg/v10/orm"
)

// This migration adds system_group table with two default groups: super-admin
// and admin. The associations between the users and groups are applied using
// another table, system_user_to_group, in M:M relationship.
func init() {
	type systemUser struct {
		ID int
	}

	type systemGroup struct {
		ID          int
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
		// Create system_group and system_user_to_group tables. Enable foreign key
		// constraints and create the tables only if they don't exist.
		for _, model := range []interface{}{&systemGroup{}, &systemUserToGroup{}} {
			err := db.Model(model).CreateTable(&orm.CreateTableOptions{
				FKConstraints: true,
				IfNotExists:   true,
			})
			if err != nil {
				return err
			}
		}

		// Create super-admin group and return the primary key value.
		superAdminGroup := &systemGroup{
			Name:        "super-admin",
			Description: "This group of users can access all system components.",
		}
		_, err := db.Model(superAdminGroup).Returning("id").Insert()
		if err != nil {
			return err
		}

		// Create admin group.
		_, err = db.Model(&systemGroup{
			Name:        "admin",
			Description: "This group of users can do everything except manage user accounts.",
		}).Insert()
		if err != nil {
			return err
		}

		// Associate default user (admin) with super-admin group.
		_, err = db.Model(&systemUserToGroup{
			UserID:  1,
			GroupID: superAdminGroup.ID,
		}).Insert()
		if err != nil {
			return err
		}

		// It is hard (if possible) to do this with pure ORM. This query associates
		// all existing users except user admin with the group admin.
		_, err = db.Exec(`WITH non_admin_assoc AS (
                            SELECT id, 2 FROM system_user WHERE id != 1
                          )
                          INSERT INTO system_user_to_group SELECT * FROM non_admin_assoc`)
		if err != nil {
			return err
		}

		_, err = db.Exec(
			`-- Checks that the deleted operation does not remove last user.
                  -- It is called by the DELETE triggers on the system_user table.
                  CREATE OR REPLACE FUNCTION system_user_check_last_user()
                  RETURNS trigger
                  LANGUAGE plpgsql
                  AS $function$
                  DECLARE
                    user_count integer;
                    user_group integer;
                  BEGIN
                    SELECT group_id FROM system_user_to_group WHERE user_id = OLD.id INTO user_group;
                    IF user_group != 1 THEN
                      RETURN OLD;
                    END IF;
                    SELECT COUNT(*) FROM system_user_to_group WHERE group_id = 1 AND user_id != OLD.id INTO user_count;
                    IF user_count = 0 THEN
                      RAISE EXCEPTION 'deleting last admin user is forbidden';
                    END IF;
                    RETURN OLD;
                  END;
                  $function$;

                  -- Check last user before delete.
                  CREATE TRIGGER system_user_before_delete
                  BEFORE DELETE ON system_user
                  FOR EACH ROW EXECUTE PROCEDURE system_user_check_last_user();`)

		// All ok.
		return nil
	}, func(db migrations.DB) error {
		_, err := db.Exec(
			`-- Removes the trigger checking last user.
                  DROP TRIGGER IF EXISTS system_user_before_delete ON system_user;

                  -- Removes the check last user function.
                  DROP FUNCTION IF EXISTS system_user_check_last_user;`)

		// Drop the new tables. This also removes all associations between the
		// users and groups.
		for _, model := range []interface{}{&systemGroup{}, &systemUserToGroup{}} {
			err = db.Model(model).DropTable(&orm.DropTableOptions{
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

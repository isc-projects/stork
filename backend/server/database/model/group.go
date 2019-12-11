package dbmodel

const SuperAdminGroupId int = 1

// Represents a group of users having some specific permissions.
type SystemGroup struct {
	Id          int
	Name        string
	Description string

	Users []*SystemUser `pg:"many2many:system_user_to_group,fk:group_id,joinFK:user_id"`
}

type SystemGroups []*SystemGroup

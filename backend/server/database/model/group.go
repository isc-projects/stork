package dbmodel

type SystemGroup struct {
	Id int
	Name string
	Description string

	Users []*SystemUser `pg:"many2many:system_user_to_group,joinFK:group_id"`
}

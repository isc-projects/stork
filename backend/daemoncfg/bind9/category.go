package bind9config

var _ formattedElement = (*Category)(nil)

// Category is the statement used within the logging scope to list
// the channels used were messages from a given category are logged.
// The "category" statement has the following format:
//
//	category <name> { <channel> ... };
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#namedconf-statement-category.
type Category struct {
	// Category name.
	Name String `parser:"@@"`
	// List of channel names where messages of this category are logged.
	Channels []String `parser:"'{' ( @@ ';'* )* '}'"`
}

// Returns the serialized BIND 9 configuration for the category statement.
func (c *Category) getFormattedOutput(filter *Filter) formatterOutput {
	categoryClause := newFormatterClause()
	categoryClause.addToken("category")
	categoryClause.addToken(c.Name.GetOriginalValue())
	categoryClauseScope := newFormatterScope()
	for _, channel := range c.Channels {
		categoryClauseScope.add(newFormatterClause(channel.GetOriginalValue()))
	}
	categoryClause.add(categoryClauseScope)
	return categoryClause
}

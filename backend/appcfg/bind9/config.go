package bind9config

import (
	"path/filepath"

	"github.com/pkg/errors"
)

// Returns the view with the given name or nil if the view is not found.
func (c *Config) GetView(viewName string) *View {
	for _, statement := range c.Statements {
		if statement.View != nil && statement.View.Name == viewName {
			return statement.View
		}
	}
	return nil
}

// Returns the key with the given name or nil if the key is not found.
func (c *Config) GetKey(keyID string) *Key {
	for _, statement := range c.Statements {
		if statement.Key != nil && statement.Key.Name == keyID {
			return statement.Key
		}
	}
	return nil
}

// Returns the ACL with the given name or nil if the ACL is not found.
func (c *Config) GetACL(aclName string) *ACL {
	for _, statement := range c.Statements {
		if statement.ACL != nil && statement.ACL.Name == aclName {
			return statement.ACL
		}
	}
	return nil
}

// Recursively searches for a key in the address-match-list. If the list
// contains references to other ACLs, it searches for a key in the referenced
// ACLs. It protects against infinite recursion by limiting the depth of the
// search to 5 levels.
func (c *Config) getKeyFromAddressMatchList(level int, addressMatchList *AddressMatchList) (*Key, error) {
	if level > 5 {
		// Too much recursion.
		return nil, errors.New("too much recursion in address-match-list")
	}
	for _, element := range addressMatchList.Elements {
		switch {
		case !element.IsMatchExpected():
			// Skip the element that contains negation (!).
			continue
		case element.KeyID != "":
			// Find a key by specified name.
			return c.GetKey(element.KeyID), nil
		case element.ACL != nil:
			// Recursively search for a key in the inline ACL.
			return c.getKeyFromAddressMatchList(level+1, element.ACL.AdressMatchList)
		case element.ACLName != "":
			// Recursively search for a key in the referenced ACL.
			acl := c.GetACL(element.ACLName)
			if acl != nil {
				return c.getKeyFromAddressMatchList(level+1, acl.AdressMatchList)
			}
		default:
			continue
		}
	}
	// The key was not found.
	return nil, nil
}

// Returns the key associated with the given view or nil if the view is not found.
// The key can be associated with the view via match-clients clause and the global
// ACLs.
func (c *Config) GetViewKey(viewName string) (*Key, error) {
	view := c.GetView(viewName)
	if view == nil {
		return nil, nil
	}
	for _, clause := range view.Clauses {
		if clause.MatchClients != nil {
			return c.getKeyFromAddressMatchList(0, clause.MatchClients.AdressMatchList)
		}
	}
	return nil, nil
}

// Returns true if the ACL element contains negation (!).
func (el AddressMatchListElement) IsMatchExpected() bool {
	return !el.Negation
}

// Returns the algorithm and secret from the given key.
func (key *Key) GetAlgorithmSecret() (algorithm string, secret string, err error) {
	for _, clause := range key.Clauses {
		if clause.Algorithm != "" {
			algorithm = clause.Algorithm
		}
		if clause.Secret != "" {
			secret = clause.Secret
		}
	}
	if algorithm == "" || secret == "" {
		err = errors.Errorf("no algorithm or secret found in key %s", key.Name)
	}
	return
}

// Expands the configuration by including the contents of the included files.
// The baseDir is a path prepended to the path of the included files when their
// paths are relative.
func (c *Config) Expand(baseDir string) (*Config, error) {
	expanded := &Config{
		sourcePath: c.sourcePath,
	}
	// Go over the top-level statements and identify the include statements.
	for _, statement := range c.Statements {
		if statement.Include != nil {
			// Found an include statement.
			path := statement.Include.Path
			if !filepath.IsAbs(path) {
				// Use the absolute path to the config file.
				path = filepath.Join(baseDir, path)
			}
			// Clean the path so it may be compared with the source file path to
			// avoid the cycles.
			path = filepath.Clean(path)
			if path == c.sourcePath {
				// If the included file points to the including file, skip expanding it.
				// One could consider returning an error but we want the parser to be
				// liberal. Stork wants to be able to look into the file contents rather
				// than validate it.
				expanded.Statements = append(expanded.Statements, statement)
				continue
			}
			// Parse the included file.
			parsedInclude, err := ParseFile(path)
			if err != nil {
				return nil, err
			}
			// Append the parsed statements to the parent file.
			expanded.Statements = append(expanded.Statements, parsedInclude.Statements...)
		} else {
			// This is not an include statement. Append it as is.
			expanded.Statements = append(expanded.Statements, statement)
		}
	}
	return expanded, nil
}

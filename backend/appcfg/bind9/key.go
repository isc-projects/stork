package bind9config

import "github.com/pkg/errors"

var (
	_ formattedElement = (*Key)(nil)
	_ formattedElement = (*KeyClause)(nil)
)

// Key is the statement used to define an algorithm and secret. It has the following
// format:
//
//	key <name> key <string> {
//		algorithm <string>;
//		secret <string>;
//	};
//
// See: https://bind9.readthedocs.io/en/latest/reference.html#key-block-grammar.
type Key struct {
	// The name of the key statement.
	Name string `parser:"( @String | @Ident )"`
	// The list of clauses: an algorithm and secret. Note that they are defined
	// a list (rather than explicitly) because they can be defined in any order.
	Clauses []*KeyClause `parser:"'{' ( @@ ';'* )* '}'"`
}

// Returns the algorithm and secret from the given key.
func (key *Key) GetAlgorithmSecret() (algorithm *string, secret *string, err error) {
	for _, clause := range key.Clauses {
		if clause.Algorithm != "" {
			algorithm = &clause.Algorithm
		}
		if clause.Secret != "" {
			secret = &clause.Secret
		}
	}
	if algorithm == nil || secret == nil {
		err = errors.Errorf("no algorithm or secret found in key %s", key.Name)
	}
	return
}

// Returns the serialized BIND 9 configuration for the key statement.
func (key *Key) getFormattedOutput(filter *Filter) formatterOutput {
	keyClause := newFormatterClausef(`key "%s"`, key.Name)
	keyClauseScope := newFormatterScope()
	for _, clause := range key.Clauses {
		keyClauseScope.add(clause.getFormattedOutput(filter))
	}
	keyClause.add(keyClauseScope)
	return keyClause
}

// KeyClause is a single clause of a key statement: an algorithm or secret.
type KeyClause struct {
	// The algorithm clause.
	Algorithm string `parser:"'algorithm' ( @Ident | @String )"`
	// The secret clause.
	Secret string `parser:"| 'secret' ( @Ident | @String )"`
}

// Returns the serialized BIND 9 configuration for the key clause.
func (k *KeyClause) getFormattedOutput(filter *Filter) formatterOutput {
	clause := newFormatterClause()
	if k.Algorithm != "" {
		clause.addTokenf(`algorithm "%s"`, k.Algorithm)
	} else {
		clause.addTokenf(`secret "%s"`, k.Secret)
	}
	return clause
}

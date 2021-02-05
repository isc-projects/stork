package dbmodel

import (
	"errors"

	"github.com/go-pg/pg/v9"
	pkgerrors "github.com/pkg/errors"
)

// Secret types.
const (
	SecretCAKey       = "cakey"
	SecretCACert      = "cacert"
	SecretServerKey   = "srvkey"
	SecretServerCert  = "srvcert"
	SecretServerToken = "srvtkn"
)

// Represents an.
type Secret struct {
	Name    string
	Content string
}

// Generate new serial number from database.
func GetNewCertSerialNumber(db *pg.DB) (int64, error) {
	var certSerialNumber int64 = 0
	_, err := db.QueryOne(pg.Scan(&certSerialNumber), "SELECT nextval('certs_serial_number_seq')")
	return certSerialNumber, err
}

// Get named secret from database.
func GetSecret(db *pg.DB, name string) ([]byte, error) {
	scrt := Secret{}
	q := db.Model(&scrt)
	q = q.Where("secret.name = ?", name)
	err := q.Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem with getting secret by name: %s", name)
	}
	return []byte(scrt.Content), nil
}

// Set secret in database under given name.
func SetSecret(db *pg.DB, name string, content []byte) error {
	scrt := &Secret{
		Name:    name,
		Content: string(content),
	}
	q := db.Model(scrt)
	q = q.OnConflict("(name) DO UPDATE")
	q = q.Set("content = EXCLUDED.content")
	_, err := q.Insert()
	return err
}

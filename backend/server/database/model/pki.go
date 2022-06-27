package dbmodel

import (
	"errors"

	"github.com/go-pg/pg/v10"
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

// Structure holding named secret.
type Secret struct {
	Name    string
	Content string
}

// Generate new serial number from database.
func GetNewCertSerialNumber(db *pg.DB) (int64, error) {
	var certSerialNumber int64
	_, err := db.QueryOne(pg.Scan(&certSerialNumber), "SELECT nextval('certs_serial_number_seq')")
	return certSerialNumber, err
}

// Get named secret from database.
func GetSecret(db *pg.DB, name string) ([]byte, error) {
	secret := Secret{}
	q := db.Model(&secret)
	q = q.Where("secret.name = ?", name)
	err := q.Select()
	if errors.Is(err, pg.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, pkgerrors.Wrapf(err, "problem getting secret by name: %s", name)
	}
	return []byte(secret.Content), nil
}

// Set secret in database under given name.
func SetSecret(db *pg.DB, name string, content []byte) error {
	secret := &Secret{
		Name:    name,
		Content: string(content),
	}
	q := db.Model(secret)
	q = q.OnConflict("(name) DO UPDATE")
	q = q.Set("content = EXCLUDED.content")
	_, err := q.Insert()
	return err
}

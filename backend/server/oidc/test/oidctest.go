package oidctest

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/coreos/go-oidc/v3/oidc/oidctest"
)

// Helper function preparing test OIDC server which allows to test OIDC discovery
// and token exchange.
// It returns server URL as string which should be used as OIDC issuer URL,
// test server teardown function and an error if such occurred while generating
// RSA key.
func PrepareTestOIDCServer() (string, func(), error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", nil, err
	}
	s := &oidctest.Server{
		PublicKeys: []oidctest.PublicKey{
			{
				PublicKey: priv.Public(),
				KeyID:     "test-key",
				Algorithm: oidc.RS256,
			},
		},
	}
	var serverURL string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			rawClaims := `{
				"iss": "` + serverURL + `",
				"aud": "clientID",
				"sub": "foo",
				"exp": ` + time.Now().Add(time.Hour).Format("1136239445") + `,
				"email": "foo@example.org",
				"email_verified": true,
				"nonce": "test-nonce",
				"groups": ["stork-users", "stork-super-admins"]
			}`
			token := oidctest.SignIDToken(priv, "test-key", oidc.RS256, rawClaims)
			resp := map[string]any{
				"access_token": "fake-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
				"id_token":     token,
			}
			w.Header().Set("Content-Type", "application/json")
			err = json.NewEncoder(w).Encode(resp)
			if err != nil {
				http.Error(w, "err encoding test json", http.StatusInternalServerError)
			}
			return
		default:
			s.ServeHTTP(w, r)
		}
	})
	srv := httptest.NewServer(handler)
	serverURL = srv.URL
	s.SetIssuer(serverURL)
	return serverURL, srv.Close, nil
}

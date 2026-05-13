package oidc

type Settings struct {
	IssuerURL string `long:"oidc-issuer-url" description:"The OID Provider Issuer URL used for OIDC discovery process." env:"STORK_OIDC_ISSUER_URL" default:""`
}

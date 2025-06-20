package utils

import "crypto/tls"

type DomainProvider interface {
	IsValid(host string) bool
	Put(domain string, active bool)
	Delete(domain string)
}

type AcmeChallengeProvider interface {
	Get(domain, key string) string
	Put(domain, key, value string)
	Delete(domain, key string)
}

type CertProvider interface {
	GetCertForDomain(domain string) *tls.Certificate
}

package fake

import (
	"crypto/rand"
	"crypto/rsa"
	"github.com/1f349/mjwt"
	"github.com/1f349/mjwt/auth"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"time"
)

var SnakeOilProv = GenSnakeOilProv()

func GenSnakeOilProv() *mjwt.Issuer {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	ks := mjwt.NewKeyStore()
	keyId := uuid.NewString()
	ks.LoadPrivateKey(keyId, key)
	issuer, err := mjwt.NewIssuerWithKeyStore("violet.test", keyId, new(jwt.SigningMethodEd25519), ks)
	if err != nil {
		panic(err)
	}
	return issuer
}

func GenSnakeOilKey(perm string) string {
	p := auth.NewPermStorage()
	p.Set(perm)
	val, err := SnakeOilProv.GenerateJwt("abc", "abc", nil, 5*time.Minute, auth.AccessTokenClaims{Perms: p})
	if err != nil {
		panic(err)
	}
	return val
}

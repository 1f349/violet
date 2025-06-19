package domains

import (
	"context"
	"github.com/1f349/violet"
	"github.com/1f349/violet/database"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestDomainsNew(t *testing.T) {
	db, err := violet.InitDB("file:TestDomainsNew?mode=memory&cache=shared")
	assert.NoError(t, err)

	domains := New(context.Background(), db, 5*time.Second)
	err = db.AddDomain(context.Background(), database.AddDomainParams{Domain: "example.com", Active: true})
	assert.NoError(t, err)

	_ = domains.compile()

	if _, ok := domains.m["example.com"]; ok {
		assert.True(t, ok)
	}

	if _, ok := domains.m["www.example.com"]; !ok {
		assert.False(t, ok)
	}
}

func TestDomains_IsValid(t *testing.T) {
	// open sqlite database
	db, err := violet.InitDB("file:TestDomains_IsValid?mode=memory&cache=shared")
	assert.NoError(t, err)

	domains := New(context.Background(), db, 5*time.Second)
	err = db.AddDomain(context.Background(), database.AddDomainParams{Domain: "example.com", Active: true})
	assert.NoError(t, err)

	domains.s.Lock()
	assert.NoError(t, domains.internalCompile(domains.m))
	domains.s.Unlock()

	assert.True(t, domains.IsValid("example.com"))
	assert.True(t, domains.IsValid("www.example.com"))
	assert.False(t, domains.IsValid("notexample.com"))
	assert.False(t, domains.IsValid("www.notexample.com"))
}

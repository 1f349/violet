package domains

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDomainsNew(t *testing.T) {
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	assert.NoError(t, err)

	domains := New(db)
	_, err = db.Exec("insert into domains (domain, active) values (?, ?)", "example.com", 1)
	assert.NoError(t, err)
	domains.Compile()

	if _, ok := domains.m["example.com"]; ok {
		assert.True(t, ok)
	}

	if _, ok := domains.m["www.example.com"]; !ok {
		assert.False(t, ok)
	}
}

func TestDomains_IsValid(t *testing.T) {
	// open sqlite database
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	assert.NoError(t, err)

	domains := New(db)
	_, err = domains.db.Exec("insert into domains (domain, active) values (?, ?)", "example.com", 1)
	assert.NoError(t, err)

	domains.s.Lock()
	assert.NoError(t, domains.internalCompile(domains.m))
	domains.s.Unlock()

	assert.True(t, domains.IsValid("example.com"))
	assert.True(t, domains.IsValid("www.example.com"))
	assert.False(t, domains.IsValid("notexample.com"))
	assert.False(t, domains.IsValid("www.notexample.com"))
}

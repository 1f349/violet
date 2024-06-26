package domains

import (
	"context"
	_ "embed"
	"github.com/1f349/violet/database"
	"github.com/1f349/violet/logger"
	"github.com/1f349/violet/utils"
	"github.com/mrmelon54/rescheduler"
	"strings"
	"sync"
)

var Logger = logger.Logger.WithPrefix("Violet Domains")

// Domains is the domain list and management system.
type Domains struct {
	db *database.Queries
	s  *sync.RWMutex
	m  map[string]struct{}
	r  *rescheduler.Rescheduler
}

// New creates a new domain list
func New(db *database.Queries) *Domains {
	a := &Domains{
		db: db,
		s:  &sync.RWMutex{},
		m:  make(map[string]struct{}),
	}
	a.r = rescheduler.NewRescheduler(a.threadCompile)
	return a
}

// IsValid returns true if a domain is valid.
func (d *Domains) IsValid(host string) bool {
	domain, _, _ := utils.SplitDomainPort(host, 0)

	// read lock for safety
	d.s.RLock()
	defer d.s.RUnlock()

	// check root domains `www.example.com`, `example.com`, `com`
	for len(domain) > 0 {
		if _, ok := d.m[domain]; ok {
			return true
		}
		n := strings.IndexByte(domain, '.')
		if n == -1 {
			break
		}
		domain = domain[n+1:]
	}
	return false
}

// Compile downloads the list of domains from the database and loads them into
// memory for faster lookups.
//
// This method makes use of the rescheduler instead of just ignoring multiple
// calls.
func (d *Domains) Compile() {
	d.r.Run()
}

func (d *Domains) threadCompile() {
	// new map
	domainMap := make(map[string]struct{})

	// compile map and check errors
	err := d.internalCompile(domainMap)
	if err != nil {
		Logger.Info("Compile faile", "err", err)
		return
	}

	// lock while replacing the map
	d.s.Lock()
	d.m = domainMap
	d.s.Unlock()
}

// internalCompile is a hidden internal method for querying the database during
// the Compile() method.
func (d *Domains) internalCompile(m map[string]struct{}) error {
	Logger.Info("Updating domains from database")

	// sql or something?
	rows, err := d.db.GetActiveDomains(context.Background())
	if err != nil {
		return err
	}

	for _, i := range rows {
		m[i] = struct{}{}
	}

	// check for errors
	return nil
}

func (d *Domains) Put(domain string, active bool) {
	d.s.Lock()
	defer d.s.Unlock()
	err := d.db.AddDomain(context.Background(), database.AddDomainParams{
		Domain: domain,
		Active: active,
	})
	if err != nil {
		logger.Logger.Infof("Database error: %s\n", err)
	}
}

func (d *Domains) Delete(domain string) {
	d.s.Lock()
	defer d.s.Unlock()
	err := d.db.DeleteDomain(context.Background(), domain)
	if err != nil {
		logger.Logger.Infof("Database error: %s\n", err)
	}
}

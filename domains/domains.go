package domains

import (
	"database/sql"
	"github.com/MrMelon54/violet/utils"
	"log"
	"strings"
	"sync"
)

type Domains struct {
	db *sql.DB
	s  *sync.RWMutex
	m  map[string]struct{}
}

// New creates a new domain list
func New(db *sql.DB) *Domains {
	a := &Domains{
		db: db,
		s:  &sync.RWMutex{},
		m:  make(map[string]struct{}),
	}

	// init domains table
	_, err := a.db.Exec(`create table if not exists domains (id integer primary key autoincrement, domain varchar)`)
	if err != nil {
		log.Printf("[WARN] Failed to generate 'domains' table\n")
		return nil
	}

	// run compile to get the initial data
	a.Compile()
	return a
}

// IsValid returns true if a domain is valid.
func (d *Domains) IsValid(host string) bool {
	// remove the port
	domain, ok := utils.GetDomainWithoutPort(host)
	if !ok {
		return false
	}

	// read lock for safety
	d.s.RLock()
	defer d.s.RUnlock()

	// check root domains `www.example.com`, `example.com`, `com`
	// TODO: could be faster using indexes and cropping the string?
	n := strings.Split(domain, ".")
	for i := 0; i < len(n); i++ {
		if _, ok := d.m[strings.Join(n[i:], ".")]; ok {
			return true
		}
	}
	return false
}

// Compile downloads the list of domains from the database and loads them into
// memory for faster lookups.
//
// This method is asynchronous and uses locks for safety.
func (d *Domains) Compile() {
	// async compile magic
	go func() {
		domainMap := make(map[string]struct{})
		err := d.internalCompile(domainMap)
		if err != nil {
			log.Printf("[Domains] Compile failed: %s\n", err)
			return
		}
		// lock while replacing the map
		d.s.Lock()
		d.m = domainMap
		d.s.Unlock()
	}()
}

// internalCompile is a hidden internal method for querying the database during
// the Compile() method.
func (d *Domains) internalCompile(m map[string]struct{}) error {
	log.Println("[Domains] Updating domains from database")

	// sql or something?
	rows, err := d.db.Query("select name from domains where enabled = true")
	if err != nil {
		return err
	}
	defer rows.Close()

	// loop through rows and scan the allowed domain names
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return err
		}
		m[name] = struct{}{}
	}

	// check for errors
	return rows.Err()
}

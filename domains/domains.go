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

func New(db *sql.DB) *Domains {
	return &Domains{
		db: db,
		s:  &sync.RWMutex{},
		m:  make(map[string]struct{}),
	}
}

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
	n := strings.Split(domain, ".")
	for i := 0; i < len(n); i++ {
		if _, ok := d.m[strings.Join(n[i:], ".")]; ok {
			return true
		}
	}
	return false
}

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

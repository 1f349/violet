package router

import (
	"database/sql"
	_ "embed"
	"github.com/1f349/violet/proxy"
	"github.com/1f349/violet/target"
	"github.com/MrMelon54/rescheduler"
	"log"
	"net/http"
	"strings"
	"sync"
)

// Manager is a database and mutex wrap around router allowing it to be
// dynamically regenerated after updating the database of routes.
type Manager struct {
	db *sql.DB
	s  *sync.RWMutex
	r  *Router
	p  *proxy.HybridTransport
	z  *rescheduler.Rescheduler
}

var (
	//go:embed create-tables.sql
	createTables string
)

// NewManager create a new manager, initialises the routes and redirects tables
// in the database and runs a first time compile.
func NewManager(db *sql.DB, proxy *proxy.HybridTransport) *Manager {
	m := &Manager{
		db: db,
		s:  &sync.RWMutex{},
		r:  New(proxy),
		p:  proxy,
	}
	m.z = rescheduler.NewRescheduler(m.threadCompile)

	// init routes table
	_, err := m.db.Exec(createTables)
	if err != nil {
		log.Printf("[WARN] Failed to generate tables\n")
		return nil
	}
	return m
}

func (m *Manager) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	m.s.RLock()
	r := m.r
	m.s.RUnlock()
	r.ServeHTTP(rw, req)
}

func (m *Manager) Compile() {
	m.z.Run()
}

func (m *Manager) threadCompile() {
	// new router
	router := New(m.p)

	// compile router and check errors
	err := m.internalCompile(router)
	if err != nil {
		log.Printf("[Manager] Compile failed: %s\n", err)
		return
	}

	// lock while replacing router
	m.s.Lock()
	m.r = router
	m.s.Unlock()
}

// internalCompile is a hidden internal method for querying the database during
// the Compile() method.
func (m *Manager) internalCompile(router *Router) error {
	log.Println("[Manager] Updating routes from database")

	// sql or something?
	rows, err := m.db.Query(`SELECT source, destination, flags FROM routes WHERE active = 1`)
	if err != nil {
		return err
	}
	defer rows.Close()

	// loop through rows and scan the options
	for rows.Next() {
		var (
			src, dst string
			flags    target.Flags
		)
		err := rows.Scan(&src, &dst, &flags)
		if err != nil {
			return err
		}

		router.AddRoute(target.Route{
			Src:   src,
			Dst:   dst,
			Flags: flags.NormaliseRouteFlags(),
		})
	}

	// check for errors
	if err := rows.Err(); err != nil {
		return err
	}

	// sql or something?
	rows, err = m.db.Query(`SELECT source,destination,flags,code FROM redirects WHERE active = 1`)
	if err != nil {
		return err
	}
	defer rows.Close()

	// loop through rows and scan the options
	for rows.Next() {
		var (
			src, dst string
			flags    target.Flags
			code     int
		)
		err := rows.Scan(&src, &dst, &flags, &code)
		if err != nil {
			return err
		}

		router.AddRedirect(target.Redirect{
			Src:   src,
			Dst:   dst,
			Flags: flags.NormaliseRedirectFlags(),
			Code:  code,
		})
	}

	// check for errors
	return rows.Err()
}

func (m *Manager) GetAllRoutes(hosts []string) ([]target.RouteWithActive, error) {
	if len(hosts) < 1 {
		return []target.RouteWithActive{}, nil
	}

	var searchString strings.Builder
	searchString.WriteString("WHERE ")
	for i := range hosts {
		if i != 0 {
			searchString.WriteString(" OR ")
		}
		searchString.WriteString("source LIKE ?")
	}

	s := make([]target.RouteWithActive, 0)

	query, err := m.db.Query(`SELECT source, destination, flags, active FROM routes `+searchString.String(), hosts)
	if err != nil {
		return nil, err
	}

	for query.Next() {
		var a target.RouteWithActive
		if query.Scan(&a.Src, &a.Dst, &a.Flags, &a.Active) != nil {
			return nil, err
		}
		s = append(s, a)
	}

	return s, nil
}

func (m *Manager) InsertRoute(route target.Route) error {
	_, err := m.db.Exec(`INSERT INTO routes (source, destination, flags) VALUES (?, ?, ?) ON CONFLICT(source) DO UPDATE SET destination = excluded.destination, flags = excluded.flags, active = 1`, route.Src, route.Dst, route.Flags)
	return err
}

func (m *Manager) DeleteRoute(source string) error {
	_, err := m.db.Exec(`UPDATE routes SET active = 0 WHERE source = ?`, source)
	return err
}

func (m *Manager) GetAllRedirects(hosts []string) ([]target.RedirectWithActive, error) {
	if len(hosts) < 1 {
		return []target.RedirectWithActive{}, nil
	}

	var searchString strings.Builder
	searchString.WriteString("WHERE ")
	for i := range hosts {
		if i != 0 {
			searchString.WriteString(" OR ")
		}
		searchString.WriteString("source LIKE ?")
	}

	s := make([]target.RedirectWithActive, 0)

	query, err := m.db.Query(`SELECT source, destination, flags, code, active FROM redirects `+searchString.String(), hosts)
	if err != nil {
		return nil, err
	}

	for query.Next() {
		var a target.RedirectWithActive
		if query.Scan(&a.Src, &a.Dst, &a.Flags, &a.Code, &a.Active) != nil {
			return nil, err
		}
		s = append(s, a)
	}

	return s, nil
}

func (m *Manager) InsertRedirect(redirect target.Redirect) error {
	_, err := m.db.Exec(`INSERT INTO redirects (source, destination, flags, code) VALUES (?, ?, ?, ?) ON CONFLICT(source) DO UPDATE SET destination = excluded.destination, flags = excluded.flags, code = excluded.code, active = 1`, redirect.Src, redirect.Dst, redirect.Flags, redirect.Code)
	return err
}

func (m *Manager) DeleteRedirect(source string) error {
	_, err := m.db.Exec(`UPDATE redirects SET active = 0 WHERE source = ?`, source)
	return err
}

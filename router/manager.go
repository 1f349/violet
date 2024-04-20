package router

import (
	"context"
	_ "embed"
	"github.com/1f349/violet/database"
	"github.com/1f349/violet/proxy"
	"github.com/1f349/violet/target"
	"github.com/mrmelon54/rescheduler"
	"log"
	"net/http"
	"strings"
	"sync"
)

// Manager is a database and mutex wrap around router allowing it to be
// dynamically regenerated after updating the database of routes.
type Manager struct {
	db *database.Queries
	s  *sync.RWMutex
	r  *Router
	p  *proxy.HybridTransport
	z  *rescheduler.Rescheduler
}

// NewManager create a new manager, initialises the routes and redirects tables
// in the database and runs a first time compile.
func NewManager(db *database.Queries, proxy *proxy.HybridTransport) *Manager {
	m := &Manager{
		db: db,
		s:  &sync.RWMutex{},
		r:  New(proxy),
		p:  proxy,
	}
	m.z = rescheduler.NewRescheduler(m.threadCompile)
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
	routeRows, err := m.db.GetActiveRoutes(context.Background())
	if err != nil {
		return err
	}

	for _, row := range routeRows {
		router.AddRoute(target.Route{
			Src:   row.Source,
			Dst:   row.Destination,
			Flags: row.Flags.NormaliseRouteFlags(),
		})
	}

	// sql or something?
	redirectsRows, err := m.db.GetActiveRedirects(context.Background())
	if err != nil {
		return err
	}

	for _, row := range redirectsRows {
		router.AddRedirect(target.Redirect{
			Src:   row.Source,
			Dst:   row.Destination,
			Flags: row.Flags.NormaliseRedirectFlags(),
			Code:  row.Code,
		})
	}

	// check for errors
	return nil
}

func (m *Manager) GetAllRoutes(hosts []string) ([]target.RouteWithActive, error) {
	if len(hosts) < 1 {
		return []target.RouteWithActive{}, nil
	}

	s := make([]target.RouteWithActive, 0)

	rows, err := m.db.GetAllRoutes(context.Background())
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		a := target.RouteWithActive{
			Route: target.Route{
				Src:   row.Source,
				Dst:   row.Destination,
				Desc:  row.Description,
				Flags: row.Flags,
			},
			Active: row.Active,
		}

		for _, i := range hosts {
			// if this is never true then the domain was mistakenly grabbed from the database
			if a.OnDomain(i) {
				s = append(s, a)
				break
			}
		}
	}

	return s, nil
}

func (m *Manager) InsertRoute(route target.RouteWithActive) error {
	return m.db.AddRoute(context.Background(), database.AddRouteParams{
		Source:      route.Src,
		Destination: route.Dst,
		Description: route.Desc,
		Flags:       route.Flags,
		Active:      route.Active,
	})
}

func (m *Manager) DeleteRoute(source string) error {
	return m.db.RemoveRoute(context.Background(), source)
}

func (m *Manager) GetAllRedirects(hosts []string) ([]target.RedirectWithActive, error) {
	if len(hosts) < 1 {
		return []target.RedirectWithActive{}, nil
	}

	s := make([]target.RedirectWithActive, 0)

	rows, err := m.db.GetAllRedirects(context.Background())
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		a := target.RedirectWithActive{
			Redirect: target.Redirect{
				Src:   row.Source,
				Dst:   row.Destination,
				Desc:  row.Description,
				Flags: row.Flags,
				Code:  row.Code,
			},
			Active: row.Active,
		}

		for _, i := range hosts {
			// if this is never true then the domain was mistakenly grabbed from the database
			if a.OnDomain(i) {
				s = append(s, a)
				break
			}
		}
	}

	return s, nil
}

func (m *Manager) InsertRedirect(redirect target.RedirectWithActive) error {
	return m.db.AddRedirect(context.Background(), database.AddRedirectParams{
		Source:      redirect.Src,
		Destination: redirect.Dst,
		Description: redirect.Desc,
		Flags:       redirect.Flags,
		Code:        redirect.Code,
		Active:      redirect.Active,
	})
}

func (m *Manager) DeleteRedirect(source string) error {
	return m.db.RemoveRedirect(context.Background(), source)
}

// GenerateHostSearch this should help improve performance
// TODO(Melon) discover how to implement this correctly
func GenerateHostSearch(hosts []string) (string, []string) {
	var searchString strings.Builder
	searchString.WriteString("WHERE ")

	hostArgs := make([]string, len(hosts)*2)
	for i := range hosts {
		if i != 0 {
			searchString.WriteString(" OR ")
		}
		// these like checks are not perfect but do reduce load on the database
		searchString.WriteString("source LIKE '%' + ? + '/%'")
		searchString.WriteString(" OR source LIKE '%' + ?")

		// loads the hostname into even and odd args
		hostArgs[i*2] = hosts[i]
		hostArgs[i*2+1] = hosts[i]
	}

	return searchString.String(), hostArgs
}

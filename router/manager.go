package router

import (
	"database/sql"
	_ "embed"
	"fmt"
	"github.com/MrMelon54/rescheduler"
	"github.com/MrMelon54/violet/proxy"
	"github.com/MrMelon54/violet/target"
	"github.com/MrMelon54/violet/utils"
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
	//go:embed create-table-routes.sql
	createTableRoutes string
	//go:embed create-table-redirects.sql
	createTableRedirects string
	//go:embed query-table-routes.sql
	queryTableRoutes string
	//go:embed query-table-redirects.sql
	queryTableRedirects string
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
	_, err := m.db.Exec(createTableRoutes)
	if err != nil {
		log.Printf("[WARN] Failed to generate 'routes' table\n")
		return nil
	}

	// init redirects table
	_, err = m.db.Exec(createTableRedirects)
	if err != nil {
		log.Printf("[WARN] Failed to generate 'redirects' table\n")
		return nil
	}

	// run compile to get the initial router
	m.Compile()
	return m
}

func (m *Manager) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	m.s.RLock()
	m.r.ServeHTTP(rw, req)
	m.s.RUnlock()
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
	rows, err := m.db.Query(queryTableRoutes)
	if err != nil {
		return err
	}
	defer rows.Close()

	// loop through rows and scan the options
	for rows.Next() {
		var (
			pre, abs, cors, secure_mode, forward_host, forward_addr, ignore_cert bool
			src, dst                                                             string
		)
		err := rows.Scan(&src, &pre, &dst, &abs, &cors, &secure_mode, &forward_host, &forward_addr, &ignore_cert)
		if err != nil {
			return err
		}

		err = addRoute(router, src, dst, target.Route{
			Pre:         pre,
			Abs:         abs,
			Cors:        cors,
			SecureMode:  secure_mode,
			ForwardHost: forward_host,
			ForwardAddr: forward_addr,
			IgnoreCert:  ignore_cert,
		})
		if err != nil {
			return err
		}
	}

	// check for errors
	if err := rows.Err(); err != nil {
		return err
	}

	// sql or something?
	rows, err = m.db.Query(queryTableRedirects)
	if err != nil {
		return err
	}
	defer rows.Close()

	// loop through rows and scan the options
	for rows.Next() {
		var (
			pre, abs bool
			code     int
			src, dst string
		)
		err := rows.Scan(&src, &pre, &dst, &abs, &code)
		if err != nil {
			return err
		}

		err = addRedirect(router, src, dst, target.Redirect{
			Pre:  pre,
			Abs:  abs,
			Code: code,
		})
		if err != nil {
			return err
		}
	}

	// check for errors
	return rows.Err()
}

// addRoute is an alias to parse the src and dst then add the route
func addRoute(router *Router, src string, dst string, t target.Route) error {
	srcHost, srcPath, dstHost, dstPort, dstPath, err := parseSrcDstHost(src, dst)
	if err != nil {
		return err
	}

	// update target route values and add route
	t.Host = dstHost
	t.Port = dstPort
	t.Path = dstPath
	router.AddRoute(srcHost, srcPath, t)
	return nil
}

// addRedirect is an alias to parse the src and dst then add the redirect
func addRedirect(router *Router, src string, dst string, t target.Redirect) error {
	srcHost, srcPath, dstHost, dstPort, dstPath, err := parseSrcDstHost(src, dst)
	if err != nil {
		return err
	}

	t.Host = dstHost
	t.Port = dstPort
	t.Path = dstPath
	router.AddRedirect(srcHost, srcPath, t)
	return nil
}

// parseSrcDstHost extracts the host/path and host:port/path from the src and dst values
func parseSrcDstHost(src string, dst string) (string, string, string, int, string, error) {
	// check if source has path
	var srcHost, srcPath string
	nSrc := strings.IndexByte(src, '/')
	if nSrc == -1 {
		// set host then path to /
		srcHost = src
		srcPath = "/"
	} else {
		// set host then custom path
		srcHost = src[:nSrc]
		srcPath = src[nSrc:]
	}

	// check if destination has path
	var dstPath string
	nDst := strings.IndexByte(dst, '/')
	if nDst == -1 {
		// set path to /
		dstPath = "/"
	} else {
		// set custom path then trim dst string to the host
		dstPath = dst[nDst:]
		dst = dst[:nDst]
	}

	// try to split the destination host into domain + port
	dstHost, dstPort, ok := utils.SplitDomainPort(dst, 0)
	if !ok {
		return "", "", "", 0, "", fmt.Errorf("failed to split destination '%s' into host + port", dst)
	}

	return srcHost, srcPath, dstHost, dstPort, dstPath, nil
}

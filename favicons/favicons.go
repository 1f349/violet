package favicons

import (
	"database/sql"
	"errors"
	"fmt"
	"golang.org/x/sync/errgroup"
	"log"
	"sync"
)

var ErrFaviconNotFound = errors.New("favicon not found")

// Favicons is a dynamic favicon generator which supports overwriting favicons
type Favicons struct {
	db         *sql.DB
	cmd        string
	cLock      *sync.RWMutex
	faviconMap map[string]*FaviconList
}

// New creates a new dynamic favicon generator
func New(db *sql.DB, inkscapeCmd string) *Favicons {
	f := &Favicons{
		db:         db,
		cmd:        inkscapeCmd,
		cLock:      &sync.RWMutex{},
		faviconMap: make(map[string]*FaviconList),
	}

	// init favicons table
	_, err := f.db.Exec(`create table if not exists favicons (id integer primary key autoincrement, host varchar, svg varchar, png varchar, ico varchar)`)
	if err != nil {
		log.Printf("[WARN] Failed to generate 'favicons' table\n")
		return nil
	}

	// run compile to get the initial data
	f.Compile()
	return f
}

// Compile downloads the list of favicon mappings from the database and loads
// them and the target favicons into memory for faster lookups
func (f *Favicons) Compile() {
	// async compile magic
	go func() {
		// new map
		favicons := make(map[string]*FaviconList)

		// compile map and check errors
		err := f.internalCompile(favicons)
		if err != nil {
			// log compile errors
			log.Printf("[Favicons] Compile failed: %s\n", err)
			return
		}

		// lock while replacing the map
		f.cLock.Lock()
		f.faviconMap = favicons
		f.cLock.Unlock()
	}()
}

// GetIcons returns the favicon list for the provided host or nil if no
// icon is found or generated
func (f *Favicons) GetIcons(host string) *FaviconList {
	// read lock for safety
	f.cLock.RLock()
	defer f.cLock.RUnlock()

	// return value from map
	return f.faviconMap[host]
}

// internalCompile is a hidden internal method for loading and generating all
// favicons.
func (f *Favicons) internalCompile(faviconMap map[string]*FaviconList) error {
	// query all rows in database
	query, err := f.db.Query(`select host, svg, png, ico from favicons`)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %w", err)
	}

	// loop over rows and scan in data using error group to catch errors
	var g errgroup.Group
	for query.Next() {
		var host, rawSvg, rawPng, rawIco string
		err := query.Scan(&host, &rawSvg, &rawPng, &rawIco)
		if err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// create favicon list for this row
		l := &FaviconList{
			Ico: CreateFaviconImage(rawIco),
			Png: CreateFaviconImage(rawPng),
			Svg: CreateFaviconImage(rawSvg),
		}

		// save the favicon list to the map
		faviconMap[host] = l

		// run the pre-process in a separate goroutine
		g.Go(func() error {
			return l.PreProcess(f.convertSvgToPng)
		})
	}
	return g.Wait()
}

// convertSvgToPng calls svg2png which runs inkscape in a subprocess
func (f *Favicons) convertSvgToPng(in []byte) ([]byte, error) {
	return svg2png(f.cmd, in)
}

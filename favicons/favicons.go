package favicons

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"github.com/1f349/violet/database"
	"github.com/mrmelon54/rescheduler"
	"golang.org/x/sync/errgroup"
	"log"
	"sync"
)

var ErrFaviconNotFound = errors.New("favicon not found")

// Favicons is a dynamic favicon generator which supports overwriting favicons
type Favicons struct {
	db         *database.Queries
	cmd        string
	cLock      *sync.RWMutex
	faviconMap map[string]*FaviconList
	r          *rescheduler.Rescheduler
}

// New creates a new dynamic favicon generator
func New(db *database.Queries, inkscapeCmd string) *Favicons {
	f := &Favicons{
		db:         db,
		cmd:        inkscapeCmd,
		cLock:      &sync.RWMutex{},
		faviconMap: make(map[string]*FaviconList),
	}
	f.r = rescheduler.NewRescheduler(f.threadCompile)

	// run compile to get the initial data
	f.Compile()
	return f
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

// Compile downloads the list of favicon mappings from the database and loads
// them and the target favicons into memory for faster lookups
//
// This method makes use of the rescheduler instead of just ignoring multiple
// calls.
func (f *Favicons) Compile() {
	f.r.Run()
}

func (f *Favicons) threadCompile() {
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
}

// internalCompile is a hidden internal method for loading and generating all
// favicons.
func (f *Favicons) internalCompile(m map[string]*FaviconList) error {
	// query all rows in database
	rows, err := f.db.GetFavicons(context.Background())
	if err != nil {
		return fmt.Errorf("failed to prepare rows: %w", err)
	}

	// loop over rows and scan in data using error group to catch errors
	var g errgroup.Group
	for _, row := range rows {
		// create favicon list for this row
		l := &FaviconList{
			Ico: CreateFaviconImage(row.Ico),
			Png: CreateFaviconImage(row.Png),
			Svg: CreateFaviconImage(row.Svg),
		}

		// save the favicon list to the map
		m[row.Host] = l

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

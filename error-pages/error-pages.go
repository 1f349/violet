package error_pages

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"sync"
)

// ErrorPages stores the custom error pages and is called by the servers to
// output meaningful pages for HTTP error codes
type ErrorPages struct {
	s       *sync.RWMutex
	m       map[int]func(rw http.ResponseWriter)
	generic func(rw http.ResponseWriter, code int)
	dir     fs.FS
}

func New(dir fs.FS) *ErrorPages {
	return &ErrorPages{
		s: &sync.RWMutex{},
		m: make(map[int]func(rw http.ResponseWriter)),
		generic: func(rw http.ResponseWriter, code int) {
			a := http.StatusText(code)
			if a != "" {
				http.Error(rw, fmt.Sprintf("%d %s\n", code, a), code)
				return
			}
			http.Error(rw, fmt.Sprintf("%d Unknown Error Code\n", code), code)
		},
		dir: dir,
	}
}

func (e *ErrorPages) ServeError(rw http.ResponseWriter, code int) {
	e.s.RLock()
	defer e.s.RUnlock()
	if p, ok := e.m[code]; ok {
		p(rw)
		return
	}
	e.generic(rw, code)
}

func (e *ErrorPages) Compile() {
	// async compile magic
	go func() {
		// new map
		errorPageMap := make(map[int]func(rw http.ResponseWriter))

		// compile map and check errors
		err := e.internalCompile(errorPageMap)
		if err != nil {
			log.Printf("[Certs] Compile failed: %s\n", err)
			return
		}
		// lock while replacing the map
		e.s.Lock()
		e.m = errorPageMap
		e.s.Unlock()
	}()
}

func (e *ErrorPages) internalCompile(m map[int]func(rw http.ResponseWriter)) error {
	// try to read dir
	files, err := fs.ReadDir(e.dir, "")
	if err != nil {
		return fmt.Errorf("failed to read error pages dir: %w", err)
	}

	log.Printf("[ErrorPages] Compiling lookup table for %d error pages\n", len(files))

	// find and load error pages
	for _, i := range files {
		// skip dirs
		if i.IsDir() {
			continue
		}

		// get file name and extension
		name := i.Name()
		ext := filepath.Ext(name)
		if ext != "html" {
			log.Printf("[ErrorPages] WARNING: non '.html' file in error pages directory: '%s'\n", name)
			continue
		}

		// try to read html file
		htmlData, err := fs.ReadFile(e.dir, name)
		if err != nil {
			return fmt.Errorf("failed to read html file '%s': %w", name, err)
		}

		// TODO: save to map
	}

	// well no errors happened
	return nil
}

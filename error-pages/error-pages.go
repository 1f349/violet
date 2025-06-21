package error_pages

import (
	"fmt"
	"github.com/1f349/violet/logger"
	"io/fs"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var Logger = logger.Logger.WithPrefix("Violet Error Pages")

// ErrorPages stores the custom error pages and is called by the servers to
// output meaningful pages for HTTP error codes
type ErrorPages struct {
	s       *sync.RWMutex
	m       map[int]func(rw http.ResponseWriter)
	generic func(rw http.ResponseWriter, code int)
	dir     fs.FS
}

// New creates a new error pages generator
func New(dir fs.FS) (*ErrorPages, error) {
	e := &ErrorPages{
		s: &sync.RWMutex{},
		m: make(map[int]func(rw http.ResponseWriter)),
		// generic error page writer
		generic: func(rw http.ResponseWriter, code int) {
			// if status text is empty then the code is unknown
			a := http.StatusText(code)
			fmt.Printf("%d - %s\n", code, a)
			if a != "" {
				// output in "xxx Error Text" format
				http.Error(rw, fmt.Sprintf("%d %s\n", code, a), code)
				return
			}
			// output the code and generic unknown message
			http.Error(rw, fmt.Sprintf("%d Unknown Error Code\n", code), code)
		},
		dir: dir,
	}
	return e, e.threadCompile()
}

// ServeError writes the error page for the given code to the response writer
func (e *ErrorPages) ServeError(rw http.ResponseWriter, code int) {
	// read lock for safety
	e.s.RLock()
	defer e.s.RUnlock()

	// use the custom error page if it exists
	if p, ok := e.m[code]; ok {
		p(rw)
		return
	}

	// otherwise use the generic error page
	e.generic(rw, code)
}

func (e *ErrorPages) threadCompile() error {
	// new map
	errorPageMap := make(map[int]func(rw http.ResponseWriter))

	// compile map and check errors
	if e.dir != nil {
		err := e.internalCompile(errorPageMap)
		if err != nil {
			Logger.Info("Compile failed", "err", err)
			return err
		}
	}

	// lock while replacing the map
	e.s.Lock()
	e.m = errorPageMap
	e.s.Unlock()

	return nil
}

func (e *ErrorPages) internalCompile(m map[int]func(rw http.ResponseWriter)) error {
	// try to read dir
	files, err := fs.ReadDir(e.dir, ".")
	if err != nil {
		return fmt.Errorf("failed to read error pages dir: %w", err)
	}

	Logger.Info("Compiling lookup table", "page count", len(files))

	// find and load error pages
	for _, i := range files {
		// skip dirs
		if i.IsDir() {
			continue
		}

		// get file name and extension
		name := i.Name()
		ext := filepath.Ext(name)

		// if the extension is not 'html' then ignore the file
		if ext != ".html" {
			Logger.Warn("Ignoring non '.html' file in error pages directory", "name", name)
			continue
		}

		// if the name can't be
		nameInt, err := strconv.Atoi(strings.TrimSuffix(name, ".html"))
		if err != nil {
			Logger.Warn("Ignoring invalid error page in error pages directory", "name", name)
			continue
		}

		// check if code is in range 100-599
		if nameInt < 100 || nameInt >= 600 {
			Logger.Warn("Ignoring invalid error page in error pages directory must be 100-599", "name", name)
			continue
		}

		// try to read html file
		htmlData, err := fs.ReadFile(e.dir, name)
		if err != nil {
			return fmt.Errorf("failed to read html file '%s': %w", name, err)
		}

		// create a callback function to write the page
		m[nameInt] = func(rw http.ResponseWriter) {
			rw.Header().Set("Content-Type", "text/html; encoding=utf-8")
			rw.WriteHeader(nameInt)
			_, _ = rw.Write(htmlData)
		}
	}

	// well no errors happened
	return nil
}

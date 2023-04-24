package favicons

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/mrmelon54/png2ico"
	"golang.org/x/sync/errgroup"
	"image/png"
	"io"
	"log"
	"net/http"
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

// FaviconList contains the ico, png and svg icons for separate favicons
type FaviconList struct {
	Ico *FaviconImage // can be generated from png with wrapper
	Png *FaviconImage // can be generated from svg with inkscape
	Svg *FaviconImage
}

// ProduceIco outputs the bytes of the ico icon or an error
func (l *FaviconList) ProduceIco() ([]byte, error) {
	if l.Ico == nil {
		return nil, ErrFaviconNotFound
	}
	return l.Ico.Raw, nil
}

// ProducePng outputs the bytes of the png icon or an error
func (l *FaviconList) ProducePng() ([]byte, error) {
	if l.Png == nil {
		return nil, ErrFaviconNotFound
	}
	return l.Png.Raw, nil
}

// ProduceSvg outputs the bytes of the svg icon or an error
func (l *FaviconList) ProduceSvg() ([]byte, error) {
	if l.Svg == nil {
		return nil, ErrFaviconNotFound
	}
	return l.Svg.Raw, nil
}

// PreProcess takes an input of the svg2png conversion function and outputs
// an error if the SVG, PNG or ICO fails to download or generate
func (l *FaviconList) PreProcess(convert func(in []byte) ([]byte, error)) error {
	var err error

	// SVG
	if l.Svg != nil {
		// download SVG
		l.Svg.Raw, err = getFaviconViaRequest(l.Svg.Url)
		if err != nil {
			return fmt.Errorf("[Favicons] Failed to fetch SVG icon: %w", err)
		}
		l.Svg.Hash = hex.EncodeToString(sha256.New().Sum(l.Svg.Raw))
	}

	// PNG
	if l.Png != nil {
		// download PNG
		l.Png.Raw, err = getFaviconViaRequest(l.Png.Url)
		if err != nil {
			return fmt.Errorf("[Favicons] Failed to fetch PNG icon: %w", err)
		}
	} else if l.Svg != nil {
		// generate PNG from SVG
		l.Png = &FaviconImage{}
		l.Png.Raw, err = convert(l.Svg.Raw)
		if err != nil {
			return fmt.Errorf("[Favicons] Failed to generate PNG icon: %w", err)
		}
	}

	// ICO
	if l.Ico != nil {
		// download ICO
		l.Ico.Raw, err = getFaviconViaRequest(l.Ico.Url)
		if err != nil {
			return fmt.Errorf("[Favicons] Failed to fetch ICO icon: %w", err)
		}
	} else if l.Png != nil {
		// generate ICO from PNG
		l.Ico = &FaviconImage{}
		decode, err := png.Decode(bytes.NewReader(l.Png.Raw))
		if err != nil {
			return fmt.Errorf("[Favicons] Failed to decode PNG icon: %w", err)
		}
		b := decode.Bounds()
		l.Ico.Raw, err = png2ico.ConvertPngToIco(l.Png.Raw, b.Dx(), b.Dy())
		if err != nil {
			return fmt.Errorf("[Favicons] Failed to generate ICO icon: %w", err)
		}
	}

	// generate sha256 hashes for svg, png and ico
	l.genSha256()
	return nil
}

// genSha256 generates sha256 hashes
func (l *FaviconList) genSha256() {
	if l.Svg != nil {
		l.Svg.Hash = genSha256(l.Svg.Raw)
	}
	if l.Png != nil {
		l.Png.Hash = genSha256(l.Png.Raw)
	}
	if l.Ico != nil {
		l.Ico.Hash = genSha256(l.Ico.Raw)
	}
}

// getFaviconViaRequest uses the standard http request library to download
// icons, outputs the raw bytes from the download or an error.
func getFaviconViaRequest(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("[Favicons] Failed to send request '%s': %w", url, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("[Favicons] Failed to do request '%s': %w", url, err)
	}
	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("[Favicons] Failed to read response '%s': %w", url, err)
	}
	return rawBody, nil
}

// genSha256 generates a sha256 hash as a hex encoded string
func genSha256(in []byte) string {
	// create sha256 generator and write to it
	h := sha256.New()
	_, err := h.Write(in)
	if err != nil {
		return ""
	}
	// encode as hex
	return hex.EncodeToString(h.Sum(nil))
}

// FaviconImage stores the url, hash and raw bytes of an image
type FaviconImage struct {
	Url  string
	Hash string
	Raw  []byte
}

// CreateFaviconImage outputs a FaviconImage with the specified URL or nil if
// the URL is an empty string.
func CreateFaviconImage(url string) *FaviconImage {
	if url == "" {
		return nil
	}
	return &FaviconImage{Url: url}
}

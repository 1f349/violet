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
		log.Printf("[WARN] Failed to generate 'domains' table\n")
		return nil
	}

	// run compile to get the initial data
	f.Compile()
	return f
}

type Favicons struct {
	db         *sql.DB
	cmd        string
	cLock      *sync.RWMutex
	faviconMap map[string]*FaviconList
}

func (f *Favicons) Compile() {
	go func() {
		favicons := make(map[string]*FaviconList)
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

func (f *Favicons) GetIcons(host string) (*FaviconList, bool) {
	f.cLock.RLock()
	defer f.cLock.RUnlock()
	if a, ok := f.faviconMap[host]; ok {
		return a, true
	}
	return nil, false
}

func (f *Favicons) internalCompile(faviconMap map[string]*FaviconList) error {
	// query all rows in database
	query, err := f.db.Query(`select * from favicons`)
	if err != nil {
		return fmt.Errorf("failed to prepare query: %w", err)
	}

	var g errgroup.Group
	for query.Next() {
		var host, rawSvg, rawPng, rawIco string
		err := query.Scan(&host, &rawSvg, &rawPng, &rawIco)
		if err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		l := &FaviconList{
			Ico: CreateFaviconImage(rawIco),
			Png: CreateFaviconImage(rawPng),
			Svg: CreateFaviconImage(rawSvg),
		}
		faviconMap[host] = l
		g.Go(func() error {
			return l.PreProcess(f.convertSvgToPng)
		})
	}
	return g.Wait()
}

func (f *Favicons) convertSvgToPng(in []byte) ([]byte, error) {
	return svg2png(f.cmd, in)
}

type FaviconList struct {
	Ico *FaviconImage // can be generated from png with wrapper
	Png *FaviconImage // can be generated from svg with inkscape
	Svg *FaviconImage
}

func (l *FaviconList) ProduceIco() ([]byte, error) {
	if l.Ico == nil {
		return nil, ErrFaviconNotFound
	}
	return l.Ico.Raw, nil
}

func (l *FaviconList) ProducePng() ([]byte, error) {
	if l.Png == nil {
		return nil, ErrFaviconNotFound
	}
	return l.Png.Raw, nil
}

func (l *FaviconList) ProduceSvg() ([]byte, error) {
	if l.Svg == nil {
		return nil, ErrFaviconNotFound
	}
	return l.Svg.Raw, nil
}

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
	l.genSha256()
	return nil
}

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

func genSha256(in []byte) string {
	h := sha256.New()
	_, err := h.Write(in)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

type FaviconImage struct {
	Url  string
	Hash string
	Raw  []byte
}

func CreateFaviconImage(url string) *FaviconImage {
	if url == "" {
		return nil
	}
	return &FaviconImage{Url: url}
}

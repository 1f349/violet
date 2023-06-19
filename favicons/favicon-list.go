package favicons

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/MrMelon54/png2ico"
	"image/png"
	"io"
	"net/http"
)

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
var getFaviconViaRequest = func(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("[Favicons] Failed to send request '%s': %w", url, err)
	}
	req.Header.Set("X-Violet-Raw-Favicon", "1")
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

package favicons

import "database/sql"

// FaviconImage stores the url, hash and raw bytes of an image
type FaviconImage struct {
	Url  string
	Hash string
	Raw  []byte
}

// CreateFaviconImage outputs a FaviconImage with the specified URL or nil if
// the URL is an empty string.
func CreateFaviconImage(url sql.NullString) *FaviconImage {
	if !url.Valid {
		return nil
	}
	return &FaviconImage{Url: url.String}
}

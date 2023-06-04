package favicons

import (
	"bytes"
	"database/sql"
	_ "embed"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	//go:embed example.svg
	exampleSvg []byte
	//go:embed example.png
	examplePng []byte
	//go:embed example.ico
	exampleIco []byte
)

func TestFaviconsNew(t *testing.T) {
	getFaviconViaRequest = func(_ string) ([]byte, error) { return exampleSvg, nil }

	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	assert.NoError(t, err)

	favicons := New(db, "inkscape")
	_, err = db.Exec("insert into favicons (host, svg, png, ico) values (?, ?, ?, ?)", "example.com", "https://example.com/assets/logo.svg", "", "")
	assert.NoError(t, err)
	favicons.cLock.Lock()
	assert.NoError(t, favicons.internalCompile(favicons.faviconMap))
	favicons.cLock.Unlock()

	icons := favicons.GetIcons("example.com")
	iconSvg, err := icons.ProduceSvg()
	assert.NoError(t, err)
	iconPng, err := icons.ProducePng()
	assert.NoError(t, err)
	iconIco, err := icons.ProduceIco()
	assert.NoError(t, err)

	assert.Equal(t, "https://example.com/assets/logo.svg", icons.Svg.Url)

	assert.Equal(t, "74cdc17d0502a690941799c327d9ca1ed042e76c784def43a42937f2eed270b4", icons.Svg.Hash)
	assert.NotEqual(t, "", icons.Png.Hash)
	assert.NotEqual(t, "", icons.Ico.Hash)
}

package favicons

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"github.com/1f349/violet"
	"github.com/1f349/violet/database"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"image/png"
	"os"
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

	db, err := violet.InitDB(os.Getenv("DB"))
	assert.NoError(t, err)

	favicons := New(db, "inkscape")
	err = db.UpdateFaviconCache(context.Background(), database.UpdateFaviconCacheParams{
		Host: "example.com",
		Svg: sql.NullString{
			String: "https://example.com/assets/logo.svg",
			Valid:  true,
		},
	})
	assert.NoError(t, err)
	favicons.cLock.Lock()
	assert.NoError(t, favicons.internalCompile(favicons.faviconMap))
	favicons.cLock.Unlock()

	icons := favicons.GetIcons("example.com")
	assert.Equal(t, "https://example.com/assets/logo.svg", icons.Svg.Url)

	assert.Equal(t, "74cdc17d0502a690941799c327d9ca1ed042e76c784def43a42937f2eed270b4", icons.Svg.Hash)
	assert.NotEqual(t, "", icons.Png.Hash)
	assert.NotEqual(t, "", icons.Ico.Hash)

	// verify png bytes are a valid png image
	pngRaw := bytes.NewBuffer(icons.Png.Raw)
	_, err = png.Decode(pngRaw)
	assert.NoError(t, err)
}

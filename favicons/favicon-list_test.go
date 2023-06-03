package favicons

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFaviconList_PreProcess(t *testing.T) {
	getFaviconViaRequest = func(_ string) ([]byte, error) {
		return exampleSvg, nil
	}
	icons := &FaviconList{Svg: &FaviconImage{Url: "https://example.com/assets/logo.svg"}}
	assert.NoError(t, icons.PreProcess(func(in []byte) ([]byte, error) {
		return svg2png("inkscape", in)
	}))
	iconSvg, err := icons.ProduceSvg()
	assert.NoError(t, err)
	iconPng, err := icons.ProducePng()
	assert.NoError(t, err)
	iconIco, err := icons.ProduceIco()
	assert.NoError(t, err)

	assert.Equal(t, "https://example.com/assets/logo.svg", icons.Svg.Url)

	assert.Equal(t, "74cdc17d0502a690941799c327d9ca1ed042e76c784def43a42937f2eed270b4", icons.Svg.Hash)
	assert.Equal(t, "84841341dafbb1e54c62d160dfc5e48c3f8db4b22265a4dbe2e0318debf9b670", icons.Png.Hash)
	assert.Equal(t, "33fc667fdb0e32305f2ee27e7dd7feb781cc776638d0971db7e18cc6335a15c7", icons.Ico.Hash)

	assert.Equal(t, 0, bytes.Compare(exampleSvg, iconSvg))
	assert.Equal(t, 0, bytes.Compare(examplePng, iconPng))
	assert.Equal(t, 0, bytes.Compare(exampleIco, iconIco))
}

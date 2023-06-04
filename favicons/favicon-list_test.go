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
	assert.NotEqual(t, "", icons.Png.Hash)
	assert.NotEqual(t, "", icons.Ico.Hash)
}

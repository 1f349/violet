package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type fakeCompile struct{ done bool }

func (f *fakeCompile) Compile() {
	f.done = true
}

var _ Compilable = &fakeCompile{}

func TestMultiCompilable_Compile(t *testing.T) {
	f := &fakeCompile{}
	a := MultiCompilable{f}
	assert.False(t, f.done)
	a.Compile()
	assert.True(t, f.done)
}

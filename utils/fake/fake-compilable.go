package fake

import "github.com/1f349/violet/utils"

// Compilable implements utils.Compilable and stores if the Compile function
// is called.
type Compilable struct{ Done bool }

func (f *Compilable) Compile() { f.Done = true }

var _ utils.Compilable = &Compilable{}

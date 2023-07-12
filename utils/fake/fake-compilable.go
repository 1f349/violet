package fake

import "github.com/MrMelon54/violet/utils"

// Compilable implements utils.Compilable and stores if the Compile function
// is called.
type Compilable struct{ Done bool }

func (f *Compilable) Compile() { f.Done = true }

var _ utils.Compilable = &Compilable{}

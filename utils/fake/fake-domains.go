package fake

import "github.com/MrMelon54/violet/utils"

// Domains implements DomainProvider and makes sure `example.com` is valid
type Domains struct{}

func (f *Domains) IsValid(host string) bool { return host == "example.com" }
func (f *Domains) Put(string, bool)         {}
func (f *Domains) Delete(string)            {}
func (f *Domains) Compile()                 {}

var _ utils.DomainProvider = &Domains{}

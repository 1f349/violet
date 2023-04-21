package domains

import (
	"github.com/MrMelon54/violet/utils"
	"strings"
	"sync"
)

type Domains struct {
	s *sync.RWMutex
	m map[string]struct{}
}

func New() *Domains {
	return &Domains{
		s: &sync.RWMutex{},
		m: make(map[string]struct{}),
	}
}

func (d *Domains) IsValid(host string) bool {
	domain, ok := utils.GetDomainWithoutPort(host)
	if !ok {
		return false
	}
	d.s.RLock()
	defer d.s.RUnlock()

	n := strings.Split(domain, ".")
	for i := 0; i < len(n); i++ {
		if _, ok := d.m[strings.Join(n[i:], ".")]; ok {
			return true
		}
	}
	return false
}

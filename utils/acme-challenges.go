package utils

import "sync"

type AcmeChallenges struct {
	s *sync.RWMutex
	d map[string]*AcmeStorage
}

type AcmeStorage struct {
	s *sync.RWMutex
	v map[string]string
}

func NewAcmeChallenge() *AcmeChallenges {
	return &AcmeChallenges{
		s: &sync.RWMutex{},
		d: make(map[string]*AcmeStorage),
	}
}

func (a *AcmeChallenges) Get(domain, key string) string {
	a.s.RLock()
	defer a.s.RUnlock()
	if m := a.d[domain]; m != nil {
		m.s.RLock()
		defer m.s.RUnlock()
		return m.v[key]
	}
	return ""
}

func (a *AcmeChallenges) Put(domain, key, value string) {
	a.s.Lock()
	m := a.d[domain]
	if m == nil {
		m = &AcmeStorage{
			s: &sync.RWMutex{},
			v: make(map[string]string),
		}
		a.d[domain] = m
	}
	m.s.Lock()
	m.v[key] = value
	m.s.Unlock()
	a.s.Unlock()
}

func (a *AcmeChallenges) Delete(domain, key string) {
	a.s.Lock()
	if m := a.d[domain]; m != nil {
		delete(m.v, key)
	}
	a.s.Unlock()
}

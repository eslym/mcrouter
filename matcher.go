package main

import (
	"fmt"
	"strings"
	"sync"
)

type section[C any] struct {
	sections map[string]section[C]
	value    C
	hasValue bool
}

type matcher[C any] struct {
	sections   section[C]
	emptyValue C
	lock       sync.RWMutex
}

type Matcher[C any] interface {
	Add(pattern string, value C) error
	Remove(pattern string) bool
	Match(domain string) (C, bool)
	Contains(pattern string) bool
}

func NewMatcher[C any]() Matcher[C] {
	return &matcher[C]{
		sections: section[C]{},
	}
}

func (m *matcher[C]) Add(pattern string, value C) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	parts := strings.Split(pattern, ".")
	return m.sections.set(parts, value)
}

func (m *matcher[C]) Remove(pattern string) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	parts := strings.Split(pattern, ".")
	return m.sections.remove(parts)
}

func (m *matcher[C]) Match(domain string) (C, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	parts := strings.Split(domain, ".")
	return m.sections.match(parts, m.emptyValue)
}

func (m *matcher[C]) Contains(pattern string) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	parts := strings.Split(pattern, ".")
	sec, ok := m.sections.find(parts)
	return ok && sec.hasValue
}

func (s section[C]) match(parts []string, emptyValue C) (C, bool) {
	if len(parts) == 0 {
		if s.hasValue {
			return s.value, true
		}
		return emptyValue, false
	}
	sec, ok := s.sections[parts[len(parts)-1]]
	rest := parts[:len(parts)-1]
	if ok {
		return sec.match(rest, emptyValue)
	}
	sec, ok = s.sections["*"]
	if ok {
		return sec.match(rest, emptyValue)
	}
	sec, ok = s.sections["**"]
	if ok && s.hasValue {
		return s.value, true
	}
	return emptyValue, false
}

func (s section[C]) find(parts []string) (section[C], bool) {
	if len(parts) == 0 {
		return s, true
	}
	sec, ok := s.sections[parts[len(parts)-1]]
	rest := parts[:len(parts)-1]
	if ok {
		return sec.find(rest)
	}
	return section[C]{}, false
}

func (s section[C]) set(parts []string, value C) error {
	if len(parts) == 0 {
		if s.hasValue {
			return fmt.Errorf("pattern already exists")
		}
		s.value = value
		s.hasValue = true
		return nil
	}
	sec, ok := s.sections[parts[len(parts)-1]]
	rest := parts[:len(parts)-1]
	if ok {
		return sec.set(rest, value)
	}
	sec = section[C]{
		sections: make(map[string]section[C]),
	}
	s.sections[parts[len(parts)-1]] = sec
	return sec.set(rest, value)
}

func (s section[C]) remove(parts []string) bool {
	if len(parts) == 0 {
		if s.hasValue {
			s.hasValue = false
			return true
		}
		return false
	}
	sec, ok := s.sections[parts[len(parts)-1]]
	rest := parts[:len(parts)-1]
	if ok {
		res := sec.remove(rest)
		if res && len(sec.sections) == 0 {
			delete(s.sections, parts[len(parts)-1])
		}
		return res
	}
	return false
}

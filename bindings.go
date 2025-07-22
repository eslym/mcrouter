package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"sync"
)

type bindingManager struct {
	bindings        Matcher[McUpstream]
	connections     Map[*ssh.ServerConn, Set[string]]
	allowedBindings Map[*ssh.ServerConn, Matcher[bool]]
	lock            sync.RWMutex
}

type BindingManager interface {
	AddConnection(conn *ssh.ServerConn) error
	RemoveConnection(conn *ssh.ServerConn)
	AddBinding(conn *ssh.ServerConn, pattern string, targetPort uint32) error
	HasBinding(pattern string) bool
	Resolve(domain string) (McUpstream, bool)
	RemoveBinding(pattern string)
	SetProxyProtocol(conn *ssh.ServerConn, pattern string, proxyProtocol bool) error
	EachBinding(conn *ssh.ServerConn, callback func(upstream McUpstream) error) error
}

func NewBindingManager() BindingManager {
	return &bindingManager{
		bindings:        NewMatcher[McUpstream](),
		connections:     NewMap[*ssh.ServerConn, Set[string]](),
		allowedBindings: NewMap[*ssh.ServerConn, Matcher[bool]](),
	}
}

func (m *bindingManager) AddConnection(conn *ssh.ServerConn) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if m.connections.Contains(conn) {
		return fmt.Errorf("connection already exists")
	}
	m.connections.Set(conn, NewSet[string]())
	validator := NewMatcher[bool]()
	for domain := range conn.Permissions.Extensions {
		_ = validator.Set(domain, true)
	}
	m.allowedBindings.Set(conn, validator)
	return nil
}

func (m *bindingManager) RemoveConnection(conn *ssh.ServerConn) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if !m.connections.Contains(conn) {
		return
	}
	domains, _ := m.connections.Get(conn)
	_ = domains.Each(func(binding string) error {
		upstream, ok := m.bindings.Get(binding)
		if !ok {
			return nil
		}
		go Close(upstream)
		m.bindings.Remove(binding)
		return nil
	})
	m.connections.Remove(conn)
	m.allowedBindings.Remove(conn)
}

func (m *bindingManager) AddBinding(conn *ssh.ServerConn, pattern string, targetPort uint32) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if !m.connections.Contains(conn) {
		return fmt.Errorf("connection does not exist")
	}
	validator, _ := m.allowedBindings.Get(conn)
	if _, ok := validator.MatchPattern(pattern); !ok {
		return fmt.Errorf("binding not allowed")
	}
	if m.bindings.Contains(pattern) {
		return fmt.Errorf("binding already exists")
	}
	upstream := NewMcUpstream(pattern, conn, targetPort)
	_ = m.bindings.Set(pattern, upstream)
	domains, _ := m.connections.Get(conn)
	domains.Add(pattern)
	return nil
}

func (m *bindingManager) HasBinding(pattern string) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.bindings.Contains(pattern)
}

func (m *bindingManager) EachBinding(conn *ssh.ServerConn, callback func(upstream McUpstream) error) error {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if !m.connections.Contains(conn) {
		return nil
	}
	domains, _ := m.connections.Get(conn)
	return domains.Each(func(pattern string) error {
		upstream, _ := m.bindings.Get(pattern)
		return callback(upstream)
	})
}

func (m *bindingManager) Resolve(domain string) (McUpstream, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	upstream, ok := m.bindings.Get(domain)
	return upstream, ok
}

func (m *bindingManager) RemoveBinding(pattern string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	if !m.bindings.Contains(pattern) {
		return
	}
	upstream, _ := m.bindings.Get(pattern)
	go Close(upstream)
	m.bindings.Remove(pattern)
	domains, _ := m.connections.Get(upstream.SSHConn())
	domains.Remove(pattern)
}

func (m *bindingManager) SetProxyProtocol(conn *ssh.ServerConn, pattern string, proxyProtocol bool) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if !m.connections.Contains(conn) {
		return fmt.Errorf("connection does not exist")
	}
	if !m.bindings.Contains(pattern) {
		return fmt.Errorf("binding does not exist")
	}
	upstream, _ := m.bindings.Get(pattern)
	upstream.SetProxyProtocol(proxyProtocol)
	return nil
}

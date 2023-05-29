package main

import (
	"golang.org/x/crypto/ssh"
	"sync"
)

type bindingManager struct {
	bindings    Map[string, McUpstream]
	connections Map[*ssh.ServerConn, Set[string]]
	lock        sync.RWMutex
}

type BindingManager interface {
	AddBinding(bind string, upstream *mcUpstream)
	HasBinding(bind string) bool
	GetBinding(bind string) (McUpstream, bool)
	RemoveBinding(bind string)
	RemoveConnection(conn *ssh.ServerConn)
}

func NewBindingManager() BindingManager {
	return &bindingManager{
		bindings:    NewMap[string, McUpstream](),
		connections: NewMap[*ssh.ServerConn, Set[string]](),
	}
}

func (b *bindingManager) AddBinding(bind string, upstream *mcUpstream) {
	b.lock.Lock()
	defer b.lock.Unlock()
	b.bindings.Set(bind, upstream)
	conn := upstream.SSHConn()
	if !b.connections.Contains(conn) {
		b.connections.Set(conn, NewSet[string]())
	}
	domains, _ := b.connections.Get(conn)
	domains.Add(upstream.Domain())
}

func (b *bindingManager) HasBinding(bind string) bool {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.bindings.Contains(bind)
}

func (b *bindingManager) GetBinding(bind string) (McUpstream, bool) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.bindings.Get(bind)
}

func (b *bindingManager) RemoveBinding(bind string) {
	b.lock.Lock()
	defer b.lock.Unlock()
	upstream, ok := b.bindings.Get(bind)
	if !ok {
		return
	}
	conn := upstream.SSHConn()
	domains, _ := b.connections.Get(conn)
	domains.Remove(upstream.Domain())
	b.bindings.Remove(bind)
}

func (b *bindingManager) RemoveConnection(conn *ssh.ServerConn) {
	b.lock.Lock()
	defer b.lock.Unlock()
	domains, ok := b.connections.Get(conn)
	if !ok {
		return
	}
	_ = domains.Each(func(domain string) error {
		upstream, _ := b.bindings.Get(domain)
		_ = upstream.Close()
		b.bindings.Remove(domain)
		return nil
	})
	b.connections.Remove(conn)
}

func (b *bindingManager) GetConnectionBindings(conn *ssh.ServerConn) (Set[string], bool) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	return b.connections.Get(conn)
}

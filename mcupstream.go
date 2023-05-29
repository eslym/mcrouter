package main

import (
	"fmt"
	"golang.org/x/crypto/ssh"
	"net"
	"sync"
	"time"
)

type forwardedTCPPayload struct {
	Addr       string
	Port       uint32
	OriginAddr string
	OriginPort uint32
}

type forwardedConn struct {
	remoteAddr    net.Addr
	localAddr     net.Addr
	channel       ssh.Channel
	readDeadline  time.Time
	writeDeadline time.Time
}

type McUpstream struct {
	domain     string
	targetHost string
	targetPort uint32
	sshConn    *ssh.ServerConn
}

type McUpstreams struct {
	lock      sync.RWMutex
	upstreams map[string]*McUpstream
}

func NewMcUpstream(domain string, sshConn *ssh.ServerConn, targetHost string, targetPort uint32) *McUpstream {
	return &McUpstream{
		domain:     domain,
		sshConn:    sshConn,
		targetHost: targetHost,
		targetPort: targetPort,
	}
}

func (m *McUpstream) Domain() string {
	return m.domain
}

func (m *McUpstream) SSHConn() *ssh.ServerConn {
	return m.sshConn
}

func (m *McUpstream) Close() error {
	return m.sshConn.Close()
}

func (m *McUpstream) Dial(src net.Conn) (net.Conn, error) {
	srcHost, port, err := net.SplitHostPort(src.RemoteAddr().String())
	if err != nil {
		return nil, err
	}
	var srcPort uint32
	_, err = fmt.Sscanf(port, "%d", &srcPort)
	if err != nil {
		return nil, err
	}
	payload := forwardedTCPPayload{
		Addr:       m.targetHost,
		Port:       m.targetPort,
		OriginAddr: srcHost,
		OriginPort: srcPort,
	}
	channel, reqs, err := m.sshConn.OpenChannel("forwarded-tcpip", ssh.Marshal(&payload))
	if err != nil {
		return nil, err
	}
	go ssh.DiscardRequests(reqs)
	return &forwardedConn{
		remoteAddr: &net.TCPAddr{
			IP:   net.ParseIP(m.targetHost),
			Port: int(m.targetPort),
		},
		localAddr: src.RemoteAddr(),
		channel:   channel,
	}, nil
}

func (f *forwardedConn) Read(b []byte) (int, error) {
	if f.readDeadline.IsZero() {
		return f.channel.Read(b)
	}
	return readWithDeadline(f.channel, b, f.readDeadline)
}

func (f *forwardedConn) Write(b []byte) (int, error) {
	if f.writeDeadline.IsZero() {
		return f.channel.Write(b)
	}
	return writeWithDeadline(f.channel, b, f.writeDeadline)
}

func (f *forwardedConn) Close() error {
	err1 := f.channel.CloseWrite()
	err2 := f.channel.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

func (f *forwardedConn) LocalAddr() net.Addr {
	return f.localAddr
}

func (f *forwardedConn) RemoteAddr() net.Addr {
	return f.remoteAddr
}

func (f *forwardedConn) SetDeadline(t time.Time) error {
	f.readDeadline = t
	f.writeDeadline = t
	return nil
}

func (f *forwardedConn) SetReadDeadline(t time.Time) error {
	f.readDeadline = t
	return nil
}

func (f *forwardedConn) SetWriteDeadline(t time.Time) error {
	f.writeDeadline = t
	return nil
}

func readWithDeadline(channel ssh.Channel, buffer []byte, deadline time.Time) (int, error) {
	res := make(chan int)
	errs := make(chan error)
	go func() {
		n, err := channel.Read(buffer)
		if err != nil {
			errs <- err
		} else {
			res <- n
		}
	}()
	go func() {
		time.Sleep(time.Until(deadline))
		errs <- fmt.Errorf("read timeout")
	}()
	select {
	case n := <-res:
		return n, nil
	case err := <-errs:
		return 0, err
	}
}

func writeWithDeadline(channel ssh.Channel, buffer []byte, deadline time.Time) (int, error) {
	res := make(chan int)
	errs := make(chan error)
	go func() {
		n, err := channel.Write(buffer)
		if err != nil {
			errs <- err
		} else {
			res <- n
		}
	}()
	go func() {
		time.Sleep(time.Until(deadline))
		errs <- fmt.Errorf("write timeout")
	}()
	select {
	case n := <-res:
		return n, nil
	case err := <-errs:
		return 0, err
	}
}

func NewMcUpstreams() *McUpstreams {
	return &McUpstreams{
		upstreams: make(map[string]*McUpstream),
	}
}

func (m *McUpstreams) Add(domain string, upstream *McUpstream) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	if _, ok := m.upstreams[domain]; ok {
		return fmt.Errorf("upstream %s already exists", domain)
	}
	m.upstreams[domain] = upstream
	return nil
}

func (m *McUpstreams) Get(domain string) (*McUpstream, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	upstream, ok := m.upstreams[domain]
	return upstream, ok
}

func (m *McUpstreams) Remove(domain string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.upstreams, domain)
}

func (m *McUpstreams) Contains(domain string) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	_, ok := m.upstreams[domain]
	return ok
}

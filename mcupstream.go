package main

import (
	"fmt"
	proxyproto "github.com/pires/go-proxyproto"
	"golang.org/x/crypto/ssh"
	"net"
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
	onClose       func(self *forwardedConn)
}

type mcUpstream struct {
	closed        bool
	domain        string
	targetPort    uint32
	sshConn       *ssh.ServerConn
	connections   Set[net.Conn]
	proxyProtocol bool
}

type McUpstream interface {
	Domain() string
	SSHConn() *ssh.ServerConn
	Close() error
	CanBind(bind string) bool
	Dial(src net.Conn) (net.Conn, error)
	UseProxyProtocol() bool
	SetProxyProtocol(use bool)
}

func NewMcUpstream(domain string, sshConn *ssh.ServerConn, targetPort uint32) McUpstream {
	return &mcUpstream{
		domain:      domain,
		sshConn:     sshConn,
		targetPort:  targetPort,
		connections: NewSet[net.Conn](),
	}
}

func (m *mcUpstream) Domain() string {
	return m.domain
}

func (m *mcUpstream) SSHConn() *ssh.ServerConn {
	return m.sshConn
}

func (m *mcUpstream) Close() error {
	if m.closed {
		return nil
	}
	m.closed = true
	go closeConnections(m.connections)
	return m.sshConn.Close()
}

func (m *mcUpstream) CanBind(bind string) bool {
	_, ok := m.sshConn.Permissions.Extensions[bind]
	return ok
}

func (m *mcUpstream) UseProxyProtocol() bool {
	return m.proxyProtocol
}

func (m *mcUpstream) SetProxyProtocol(use bool) {
	m.proxyProtocol = use
}

func (m *mcUpstream) Dial(src net.Conn) (net.Conn, error) {
	if m.closed {
		return nil, fmt.Errorf("upstream closed")
	}
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
		Addr:       m.domain,
		Port:       m.targetPort,
		OriginAddr: srcHost,
		OriginPort: srcPort,
	}
	channel, reqs, err := m.sshConn.OpenChannel("forwarded-tcpip", ssh.Marshal(&payload))
	if err != nil {
		return nil, err
	}
	go ssh.DiscardRequests(reqs)
	conn := &forwardedConn{
		remoteAddr: &net.TCPAddr{
			IP:   net.IPv4zero,
			Port: 0,
		},
		localAddr: src.LocalAddr(),
		channel:   channel,
	}
	if m.proxyProtocol {
		header := proxyproto.HeaderProxyFromAddrs(1, src.RemoteAddr(), src.LocalAddr())
		_, err = header.WriteTo(conn)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
	}
	conn.onClose = func(self *forwardedConn) {
		m.connections.Remove(self)
	}
	m.connections.Add(conn)
	return conn, nil
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
	if f.onClose != nil {
		f.onClose(f)
	}
	return f.channel.Close()
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

func closeConnections(connections Set[net.Conn]) {
	_ = connections.Each(func(conn net.Conn) error {
		_ = conn.Close()
		return nil
	})
}

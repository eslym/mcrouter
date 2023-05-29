package main

import (
	"net"
)

type forwardConn struct {
	src  net.Conn
	dest net.Conn
}

type ForwardConn interface {
	Close()
}

func NewForwardConn(src net.Conn, dest net.Conn) ForwardConn {
	go forward(src, dest)
	return &forwardConn{
		src:  src,
		dest: dest,
	}
}

func (f forwardConn) Close() {
	_ = f.src.Close()
	_ = f.dest.Close()
}

func forward(src net.Conn, dest net.Conn) {
	closed := make(chan bool)
	go pipeTo(closed, src, dest)
	pipeTo(nil, dest, src)
	<-closed
	_ = src.Close()
	_ = dest.Close()
}

func pipeTo(closed chan bool, src net.Conn, dest net.Conn) {
	buf := make([]byte, 16384)
	for {
		n, err := src.Read(buf)
		if err != nil {
			break
		}
		_, err = dest.Write(buf[:n])
		if err != nil {
			break
		}
	}
	if closed != nil {
		closed <- true
	}
}

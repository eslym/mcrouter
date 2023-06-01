package main

import (
	"encoding/json"
	"github.com/Tnze/go-mc/net/packet"
	"log"
	"net"
	"time"
)

const (
	ActionLogin = 2
)

type McMessage struct {
	Text string `json:"text"`
}

var banList = NewMap[string, time.Time]()

func handleMinecraft(downstream net.Conn) {
	if tcpAddr, ok := downstream.RemoteAddr().(*net.TCPAddr); ok {
		until, ok := banList.Get(tcpAddr.IP.String())
		if ok && until.After(time.Now()) {
			_ = downstream.Close()
			if opts.LogRejected {
				log.Printf("[MC] rejected banned connection from %s, until %v", downstream.RemoteAddr().String(), until)
			}
			return
		}
	}

	p := &packet.Packet{}

	err := p.UnPack(downstream, -1)

	if err != nil {
		_ = downstream.Close()
		return
	}

	if p.ID != 0 {
		_ = downstream.Close()
		return
	}

	var (
		Version  packet.VarInt
		Host     packet.Identifier
		Port     packet.UnsignedShort
		NextStep packet.VarInt
	)

	err = p.Scan(&Version, &Host, &Port, &NextStep)

	if err != nil {
		log.Printf("[MC] Failed to parsed packet from %s: %s", downstream.RemoteAddr().String(), err.Error())
		_ = downstream.Close()
		return
	}

	if p.ID != 0 {
		log.Printf(
			"[MC] Non handshake packet received from %s",
			downstream.RemoteAddr().String(),
		)
		_ = downstream.Close()
		return
	}

	if opts.BanIP && net.ParseIP(string(Host)) != nil {
		log.Printf("[MC] %s is trying to access directly to an IP address", downstream.RemoteAddr().String())
		_ = downstream.Close()
		if tcpAddr, ok := downstream.RemoteAddr().(*net.TCPAddr); ok {
			banList.Set(tcpAddr.IP.String(), time.Now().Add(time.Duration(opts.BanDuration)*time.Hour))
		}
		return
	}

	upstream, ok := bindings.Resolve(string(Host))

	if !ok {
		action := "PING"
		if NextStep == ActionLogin {
			action = "LOGIN"
		}
		log.Printf(
			"[MC] Failed handshake from %s for %s:%d (Protocol %d, %s)",
			downstream.RemoteAddr().String(),
			Host, Port, Version, action,
		)
		if NextStep == ActionLogin {
			kick(downstream, "Server is not available")
		}
		_ = downstream.Close()
		return
	}

	upConn, err := upstream.Dial(downstream)

	if err != nil {
		log.Printf("[MC] Failed to connect upstream %s, %v", upstream.Domain(), err)
		if NextStep == ActionLogin {
			kick(downstream, "Server is not available")
		}
		_ = downstream.Close()
		return
	}

	_ = p.Pack(upConn, -1)

	forward(downstream, upConn)
}

func kick(conn net.Conn, message string) {
	m := McMessage{Text: message}
	bytes, _ := json.Marshal(m)
	pack := packet.Marshal(0x00, packet.String(bytes))
	_ = pack.Pack(conn, -1)
	time.Sleep(10 * time.Millisecond)
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

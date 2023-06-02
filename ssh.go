package main

import (
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
	"log"
	"net"
	"os"
	"path"
	"time"
)

type UserConfig struct {
	Password        string   `yaml:"password"`
	AuthorizedKeys  []string `yaml:"authorized_keys"`
	AllowedBindings []string `yaml:"allowed_bindings"`
}

func handleSSH(conn net.Conn, config *ssh.ServerConfig) {
	sshConn, channels, requests, err := ssh.NewServerConn(conn, config)

	if err != nil {
		return
	}

	_ = bindings.AddConnection(sshConn)

	go func() {
		_ = sshConn.Wait()
		bindings.RemoveConnection(sshConn)
		log.Printf("[SSH] %v disconnected", hex.EncodeToString(sshConn.SessionID()))
	}()

	log.Printf("[SSH] %v connected", hex.EncodeToString(sshConn.SessionID()))

	go handleRequests(sshConn, requests)
	go handleChannels(sshConn, channels)
	go handleKeepAlive(sshConn)
}

func handleChannels(sshConn *ssh.ServerConn, channels <-chan ssh.NewChannel) {
	for newChannel := range channels {
		switch newChannel.ChannelType() {
		case "session":
			channel, requests, err := newChannel.Accept()
			if err != nil {
				continue
			}
			go handleSession(sshConn, channel, requests)
		default:
			_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
		}
	}
}

func handleRequests(sshConn *ssh.ServerConn, requests <-chan *ssh.Request) {
	for req := range requests {
		switch req.Type {
		case "tcpip-forward":
			payload := tcpipForwardPayload{}
			err := ssh.Unmarshal(req.Payload, &payload)
			if err != nil {
				replyWith(req, false, nil)
				continue
			}
			if payload.Port == 0 {
				payload.Port = 25565
			}
			err = bindings.AddBinding(sshConn, payload.Addr, payload.Port)
			if err != nil {
				log.Printf("[SSH] binding for %v (%s:%d) is rejected: %v", hex.EncodeToString(sshConn.SessionID()), payload.Addr, payload.Port, err)
				replyWith(req, false, nil)
				continue
			}
			port := replyPort{Port: payload.Port}
			reply := ssh.Marshal(&port)
			replyWith(req, true, reply)
		default:
			replyWith(req, false, nil)
		}
	}
}

func handleKeepAlive(sshConn *ssh.ServerConn) {
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		_ = sshConn.Wait()
		ticker.Stop()
	}()
	for {
		select {
		case <-ticker.C:
			_, _, err := sshConn.SendRequest("keepalive@minecraft", true, nil)
			if err != nil {
				_ = sshConn.Close()
				return
			}
		}
	}
}

func handleSession(conn *ssh.ServerConn, channel ssh.Channel, requests <-chan *ssh.Request) {
	var pty *term.Terminal = nil
request:
	for req := range requests {
		switch req.Type {
		case "pty-req":
			if pty != nil {
				replyWith(req, false, nil)
				continue
			}
			var ptyReq = struct {
				Term    string
				Columns uint32
				Rows    uint32
				Width   uint32
				Height  uint32
				Modes   string
			}{}
			err := ssh.Unmarshal(req.Payload, &ptyReq)
			if err != nil {
				replyWith(req, false, nil)
				continue
			}
			pty = term.NewTerminal(channel, "> ")
			_ = (*pty).SetSize(int(ptyReq.Columns), int(ptyReq.Rows))
			replyWith(req, true, nil)
		case "shell", "exec":
			go func() {
				session := NewSession(conn, channel, requests, pty)
				if req.Type == "exec" {
					var cmd = struct {
						Command string
					}{}
					err := ssh.Unmarshal(req.Payload, &cmd)
					if err != nil {
						replyWith(req, false, nil)
						return
					}
					if !session.Exec(cmd.Command) {
						replyWith(req, true, nil)
						_ = channel.Close()
						return
					}
				}
				replyWith(req, true, nil)
				go session.Start()
			}()
			break request
		default:
			replyWith(req, false, nil)
		}
	}
}

func handleSSHPublicKeyAuth(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	config, err := loadConfig(conn.User())
	if err != nil {
		return nil, err
	}

	key1 := key.Marshal()

	for _, authorizedKey := range config.AuthorizedKeys {
		pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(authorizedKey))

		if err != nil {
			continue
		}

		key2 := pub.Marshal()

		if len(key1) != len(key2) {
			subtle.ConstantTimeCompare(key1, key1)
			continue
		}

		if subtle.ConstantTimeCompare(key1, key2) != 1 {
			continue
		}

		permissions := ssh.Permissions{}

		permissions.Extensions = make(map[string]string)

		for _, binding := range config.AllowedBindings {
			permissions.Extensions[binding] = binding
		}

		return userPermission(config), nil
	}

	return nil, fmt.Errorf("no matching key found")
}

func handlePasswordAuth(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	return handlePassword(conn.User(), string(password))
}

func handlePassword(username string, password string) (*ssh.Permissions, error) {
	config, err := loadConfig(username)

	if err != nil {
		return nil, err
	}

	if config.Password == "" {
		return nil, fmt.Errorf("password not set")
	}

	if len(config.Password) != len(password) {
		subtle.ConstantTimeCompare([]byte(password), []byte(password))
		return nil, fmt.Errorf("password mismatch")
	}

	if subtle.ConstantTimeCompare([]byte(config.Password), []byte(password)) == 1 {
		return userPermission(config), nil
	}

	return nil, fmt.Errorf("password mismatch")
}

func loadConfig(user string) (*UserConfig, error) {
	configPath := path.Join(opts.SSHAuth, fmt.Sprintf("%s.yaml", user))
	binary, err := os.ReadFile(configPath)

	if err != nil {
		return nil, err
	}

	var config UserConfig
	err = yaml.Unmarshal(binary, &config)

	if err != nil {
		return nil, err
	}

	return &config, nil
}

func userPermission(config *UserConfig) *ssh.Permissions {
	permissions := ssh.Permissions{}

	permissions.Extensions = make(map[string]string)

	for _, binding := range config.AllowedBindings {
		permissions.Extensions[binding] = binding
	}

	return &permissions
}

func replyWith(req *ssh.Request, ok bool, payload []byte) {
	if req.WantReply {
		_ = req.Reply(ok, payload)
	}
}

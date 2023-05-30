package main

import (
	"bytes"
	"fmt"
	"github.com/Potterli20/go-flags-fork"
	"golang.org/x/crypto/ssh"
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

type tcpipForwardPayload struct {
	Addr string
	Port uint32
}

type replyPort struct {
	Port uint32
}

type EnableProxyProtocolCommandOptions struct {
	Bindings []string `short:"b" name:"binding" description:"Bindings to enable Proxy Protocol for" required:"yes"`
}

type DisableProxyProtocolCommandOptions struct {
	Bindings []string `short:"b" name:"binding" description:"Bindings to disable Proxy Protocol for" required:"yes"`
}

type ListBindingsCommandOptions struct{}

type ExitCommandOptions struct{}

var opts struct {
	SSHListen       string `short:"S" name:"ssh" description:"SSH listen address" default:"127.0.0.1:2222"`
	MinecraftListen string `short:"M" name:"minecraft" description:"Minecraft listen address" default:"127.0.0.1:25565"`
	SSHKey          string `short:"a" name:"key" description:"SSH Server private key file" required:"yes"`
	SSHAuth         string `short:"a" name:"auth" description:"SSH Server auth directories" default:"users"`
}

var bindings BindingManager

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		return
	}

	keyBin, err := os.ReadFile(opts.SSHKey)
	if err != nil {
		log.Fatalf("Failed to read SSH private key: %v", err)
	}

	privateKey, err := ssh.ParsePrivateKey(keyBin)
	if err != nil {
		log.Fatalf("Failed to parse SSH private key: %v", err)
	}

	config := &ssh.ServerConfig{
		PublicKeyCallback: handleSSHPublicKeyAuth,
		PasswordCallback:  handlePasswordAuth,
	}

	config.AddHostKey(privateKey)

	sshListener, err := net.Listen("tcp", opts.SSHListen)

	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", opts.SSHListen, err)
	}

	minecraftListener, err := net.Listen("tcp", opts.MinecraftListen)

	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", opts.MinecraftListen, err)
	}

	bindings = NewBindingManager()

	go listenSSH(sshListener, config)
	go listenMinecraft(minecraftListener)
}

func listenSSH(listener net.Listener, config *ssh.ServerConfig) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("Failed to accept incoming connection: %v", err)
		}

		go handleSSH(conn, config)
	}
}

func listenMinecraft(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("Failed to accept incoming connection: %v", err)
		}

		go handleMinecraft(conn)
	}
}

func handleSSH(conn net.Conn, config *ssh.ServerConfig) {
	sshConn, channels, requests, err := ssh.NewServerConn(conn, config)

	defer func() {
		_ = conn.Close()
	}()

	if err != nil {
		log.Printf("Failed to handshake: %v", err)
		return
	}

	_ = bindings.AddConnection(sshConn)

	go func() {
		_ = sshConn.Wait()
		bindings.RemoveConnection(sshConn)
	}()

	go handleRequests(sshConn, requests)
	go handleChannels(sshConn, channels)
	go handleKeepAlive(sshConn)
}

func handleMinecraft(conn net.Conn) {
	// TODO: Implement
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
	ticker := time.Tick(5 * time.Second)
	for {
		select {
		case <-ticker:
			_, _, err := sshConn.SendRequest("keepalive@minecraft", true, nil)
			if err != nil {
				_ = sshConn.Close()
				return
			}
		}
	}
}

func handleSession(conn *ssh.ServerConn, channel ssh.Channel, requests <-chan *ssh.Request) {
	started := false
	for req := range requests {
		switch req.Type {
		case "shell":
			if started {
				replyWith(req, false, nil)
				continue
			}
			started = true
			replyWith(req, true, nil)
		case "exec":
			if started {
				replyWith(req, false, nil)
				continue
			}
			started = true
			replyWith(req, true, nil)
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

	for _, authorizedKey := range config.AuthorizedKeys {
		pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(authorizedKey))

		if err != nil {
			continue
		}

		if bytes.Compare(pub.Marshal(), key.Marshal()) != 0 {
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

	if config.Password == password {
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

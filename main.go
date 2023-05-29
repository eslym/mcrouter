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
		PublicKeyCallback:           handleSSHPublicKeyAuth,
		PasswordCallback:            handlePasswordAuth,
		KeyboardInteractiveCallback: handleKeyboardAuth,
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

	go handleRequests(sshConn, requests)
	go handleChannels(sshConn, channels)
	go handleKeepAlive(sshConn)
}

func handleMinecraft(conn net.Conn) {
	// TODO: Implement
}

func handleChannels(sshConn *ssh.ServerConn, channels <-chan ssh.NewChannel) {
	// TODO: Implement
}

func handleRequests(sshConn *ssh.ServerConn, requests <-chan *ssh.Request) {
	// TODO: Implement
}

func handleKeepAlive(sshConn *ssh.ServerConn) {
	ticker := time.Tick(5 * time.Second)
	for {
		select {
		case <-ticker:
			_, _, err := sshConn.SendRequest("keepalive@minecraft", true, nil)
			if err != nil {
				bindings.RemoveConnection(sshConn)
				_ = sshConn.Close()
				return
			}
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

func handleKeyboardAuth(conn ssh.ConnMetadata, client ssh.KeyboardInteractiveChallenge) (*ssh.Permissions, error) {
	user := conn.User()
	if user == "" {
		ans, err := client("", "", []string{"Username"}, []bool{true})
		if err != nil {
			return nil, err
		}
		user = ans[0]
	}
	ans, err := client("", "", []string{"Password"}, []bool{false})
	if err != nil {
		return nil, err
	}
	pass := ans[0]

	return handlePassword(user, pass)
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

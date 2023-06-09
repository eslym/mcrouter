package main

import (
	"github.com/Potterli20/go-flags-fork"
	"golang.org/x/crypto/ssh"
	"log"
	"net"
	"os"
	"time"
)

type tcpipForwardPayload struct {
	Addr string
	Port uint32
}

type replyPort struct {
	Port uint32
}

var opts struct {
	SSHListen       string   `short:"S" name:"ssh" description:"SSH listen address" default:"127.0.0.1:2222"`
	MinecraftListen string   `short:"M" name:"minecraft" description:"Minecraft listen address" default:"127.0.0.1:25565"`
	SSHKey          string   `short:"k" name:"key" description:"SSH Server private key file" required:"yes"`
	SSHAuth         string   `short:"a" name:"auth" description:"SSH Server auth directories" default:"users"`
	BanIP           bool     `short:"I" name:"ban-ip" description:"Ban IP addresses that tried to ping minecraft server directly"`
	BanDuration     uint32   `short:"D" name:"ban-duration" description:"Ban duration in hours" default:"48"`
	LogRejected     bool     `short:"R" name:"rejected" description:"Log rejected connections"`
	AllowedDomains  []string `short:"w" name:"whitelist" description:"Domain names allowed to connect"`
	DeniedDomains   []string `short:"b" name:"blacklist" description:"Domain names denied to connect"`
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

	for _, domain := range opts.AllowedDomains {
		_ = allowedDomains.Set(domain, true)
	}

	for _, domain := range opts.DeniedDomains {
		_ = deniedDomains.Set(domain, true)
	}

	log.Printf("Listening on %s for SSH Server", opts.SSHListen)
	log.Printf("Listening on %s for Minecraft Server", opts.MinecraftListen)

	bindings = NewBindingManager()

	if opts.BanIP {
		go cleanupBan()
	}

	go listenMinecraft(minecraftListener)
	listenSSH(sshListener, config)
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

func cleanupBan() {
	for {
		time.Sleep(5 * time.Minute)
		_ = banList.Filter(func(ip string, until time.Time) (bool, error) {
			return until.After(time.Now()), nil
		})
	}
}

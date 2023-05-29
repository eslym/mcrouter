package main

import (
	"github.com/Potterli20/go-flags-fork"
	"log"
)

var opts struct {
	SSHListen       string `short:"S" name:"ssh" description:"SSH listen address" default:"127.0.0.1:2222"`
	MinecraftListen string `short:"M" name:"minecraft" description:"Minecraft listen address" default:"127.0.0.1:25565"`
}

func main() {
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Fatalf("Error parsing flags: %v", err)
	}
}

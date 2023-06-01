package main

import (
	"fmt"
	"github.com/Potterli20/go-flags-fork"
	"github.com/google/shlex"
	"golang.org/x/crypto/ssh"
	"time"
)

type proxyProtoCommandOptions struct {
	Enable  []string `short:"E" name:"enable" description:"Bindings to enable Proxy Protocol for"`
	Disable []string `short:"D" name:"disable" description:"Bindings to disable Proxy Protocol for"`
}

type listCommandOptions struct {
	All bool `short:"a" name:"all" description:"Print all details"`
}

type emptyCommandOptions struct{}

type exitStatus struct {
	ExitCode uint32
}

type session struct {
	conn     *ssh.ServerConn
	channel  ssh.Channel
	signals  <-chan string
	needStop bool
}

type Session interface {
	Exec(command string) bool
	Start()
	NeedStop() bool
}

func NewSession(conn *ssh.ServerConn, channel ssh.Channel, requests <-chan *ssh.Request) Session {
	signals := make(chan string)
	go func() {
		for req := range requests {
			switch req.Type {
			case "signal":
				var signal struct {
					Name string
				}
				err := ssh.Unmarshal(req.Payload, &signal)
				if err == nil {
					signals <- signal.Name
				}
				replyWith(req, err == nil, nil)
			default:
				replyWith(req, false, nil)
			}
		}
		close(signals)
	}()
	return &session{
		conn:    conn,
		channel: channel,
		signals: signals,
	}
}

func (s *session) Exec(command string) bool {
	args, err := shlex.Split(command)
	var status exitStatus
	if err == nil {
		switch args[0] {
		case "proxy":
			err = s.handleProxyCommand(args)
		case "list":
			err = s.handleListCommand(args)
		case "help":
			err = s.handleHelpCommand(args)
		case "exit":
			err = s.handleExitCommand(args)
		default:
			err = fmt.Errorf("unknown command: %s", args[0])
		}
	}
	if err != nil {
		if flags.WroteHelp(err) {
			_, _ = fmt.Fprintln(s.channel, err)
			status.ExitCode = 0
			_, _ = s.channel.SendRequest("exit-status", false, ssh.Marshal(status))
			return false
		}
		_, _ = fmt.Fprintln(s.channel.Stderr(), err)
		status.ExitCode = 1
		_, _ = s.channel.SendRequest("exit-status", false, ssh.Marshal(status))
		return false
	}
	status.ExitCode = 0
	_, _ = s.channel.SendRequest("exit-status", false, ssh.Marshal(status))
	return true
}

func (s *session) Start() {
	defer func() {
		_ = s.channel.Close()
	}()
	_, _ = fmt.Fprint(s.channel, "mcrouter> ")
	lines := ReadLine(s.channel)
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				return
			}
			s.Exec(line)
			if s.NeedStop() {
				return
			}
			time.Sleep(5 * time.Millisecond)
			_, _ = fmt.Fprint(s.channel, "mcrouter> ")
		case signal, ok := <-s.signals:
			if !ok {
				return
			}
			switch signal {
			case "INT", "TERM", "KILL":
				fmt.Println("^C")
				return
			}
		}
	}
}

func (s *session) NeedStop() bool {
	return s.needStop
}

func (s *session) parseArgs(args []string, out any, help string) ([]string, error) {
	parser := flags.NewParser(out, flags.Default^flags.PrintErrors)
	parser.Name = args[0]
	parser.LongDescription = help
	return parser.ParseArgs(args)
}

func (s *session) handleProxyCommand(args []string) error {
	var opts proxyProtoCommandOptions
	_, err := s.parseArgs(args, &opts, "Config proxy protocol for bindings")
	if err != nil {
		return err
	}
	if len(opts.Enable) == 0 && len(opts.Disable) == 0 {
		_, _ = fmt.Fprintln(s.channel, "No bindings specified")
	}
	for _, binding := range opts.Enable {
		err = bindings.SetProxyProtocol(s.conn, binding, true)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(s.channel, "Enabled proxy protocol for", binding)
	}
	for _, binding := range opts.Disable {
		err = bindings.SetProxyProtocol(s.conn, binding, false)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(s.channel, "Enabled proxy protocol for", binding)
	}
	return nil
}

func (s *session) handleListCommand(args []string) error {
	var opts listCommandOptions
	_, err := s.parseArgs(args, &opts, "List bindings")
	if err != nil {
		return err
	}
	_ = bindings.EachBinding(s.conn, func(upstream McUpstream) error {
		_, _ = fmt.Fprint(s.channel, upstream.Domain())
		if opts.All {
			_, _ = fmt.Fprintf(
				s.channel,
				", proxy protocol:%t connections:%d\n",
				upstream.UseProxyProtocol(), upstream.GetConnections(),
			)
		} else {
			_, _ = fmt.Fprintln(s.channel)
		}
		return nil
	})
	return nil
}

func (s *session) handleExitCommand(args []string) error {
	var opts emptyCommandOptions
	_, err := s.parseArgs(args, &opts, "Exit")
	if err != nil {
		return err
	}
	s.needStop = true
	return nil
}

func (s *session) handleHelpCommand(args []string) error {
	var opts emptyCommandOptions
	_, err := s.parseArgs(args, &opts, "Show help")
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintln(s.channel, "Commands:")
	_, _ = fmt.Fprintln(s.channel, "  proxy - Config proxy protocol for bindings")
	_, _ = fmt.Fprintln(s.channel, "  list - List bindings")
	_, _ = fmt.Fprintln(s.channel, "  exit - Exit")
	return nil
}

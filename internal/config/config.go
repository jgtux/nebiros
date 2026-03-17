package config

import (
	"flag"
	"fmt"
	"os"
)

type Mode string

const (
	ModeQUIC   Mode = "quic"
	ModeMASQUE Mode = "masque"
)

type ClientConfig struct {
	Mode          Mode
	ListenAddr    string
	ServerAddr    string
	TargetAddr    string
	ProxyTemplate string
	ServerName    string
	Insecure      bool
}

type ServerConfig struct {
	Mode          Mode
	ListenAddr    string
	TargetAddr    string
	ProxyTemplate string
}

func ParseClient() ClientConfig {
	mode := flag.String("mode", string(ModeQUIC), "transport mode: quic or masque")
	listen := flag.String("listen", "127.0.0.1:51821", "local UDP listen address")
	server := flag.String("server", "127.0.0.1:4433", "remote QUIC server address")
	target := flag.String("target", "127.0.0.1:51820", "remote UDP target address")
	proxyTemplate := flag.String("proxy-template", "https://localhost:4433/masque?h={target_host}&p={target_port}", "HTTP/3 CONNECT-UDP proxy URI template")
	serverName := flag.String("server-name", "localhost", "TLS server name")
	insecure := flag.Bool("insecure", true, "skip TLS verification for self-signed certs")
	flag.Parse()

	return ClientConfig{
		Mode:          mustParseMode(*mode),
		ListenAddr:    *listen,
		ServerAddr:    *server,
		TargetAddr:    *target,
		ProxyTemplate: *proxyTemplate,
		ServerName:    *serverName,
		Insecure:      *insecure,
	}
}

func ParseServer() ServerConfig {
	mode := flag.String("mode", string(ModeQUIC), "transport mode: quic or masque")
	listen := flag.String("listen", ":4433", "transport listen address")
	target := flag.String("target", "127.0.0.1:51820", "remote UDP target address")
	proxyTemplate := flag.String("proxy-template", "https://localhost:4433/masque?h={target_host}&p={target_port}", "HTTP/3 CONNECT-UDP proxy URI template")
	flag.Parse()

	return ServerConfig{
		Mode:          mustParseMode(*mode),
		ListenAddr:    *listen,
		TargetAddr:    *target,
		ProxyTemplate: *proxyTemplate,
	}
}

func mustParseMode(v string) Mode {
	switch Mode(v) {
	case ModeQUIC, ModeMASQUE:
		return Mode(v)
	default:
		fmt.Fprintf(os.Stderr, "invalid -mode %q (expected %q or %q)\n", v, ModeQUIC, ModeMASQUE)
		os.Exit(2)
		return ""
	}
}

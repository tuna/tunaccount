package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
)

var listenAddr string

func init() {
	flag.StringVar(&listenAddr, "listen-addr", "127.0.0.1:10389", "listen address")
	flag.Parse()
}

func main() {

	server := makeLDAPServer(listenAddr)
	// When CTRL+C, SIGINT and SIGTERM signal occurs
	// Then stop server gracefully
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	close(ch)

	server.Stop()
}

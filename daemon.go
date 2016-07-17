package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gopkg.in/urfave/cli.v1"
)

func startDaemon(c *cli.Context) error {
	cfgFile := c.String("config")

	cfg, _ := loadDaemonConfig(cfgFile)

	ldapListenAddr := fmt.Sprintf("%s:%d", cfg.LDAP.ListenAddr, cfg.LDAP.ListenPort)

	server := makeLDAPServer(ldapListenAddr)

	// When CTRL+C, SIGINT and SIGTERM signal occurs
	// Then stop server gracefully
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	close(ch)

	server.Stop()
	return nil
}

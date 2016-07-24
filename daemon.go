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
	initLogger(true, true, false)

	cfg, err := loadDaemonConfig(cfgFile)
	if err != nil {
		logger.Panic(err.Error())
	}
	err = initMongo()
	if err != nil {
		logger.Panicf("Error initializing mongodb: %s", err.Error())
	}

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

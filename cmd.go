package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gopkg.in/urfave/cli.v1"
)

func prepareConfig(cfgFile string) (*DaemonConfig, error) {
	logger.Noticef("Using config file: %s", cfgFile)
	cfg, err := loadDaemonConfig(cfgFile)
	if err != nil {
		return nil, err
	}
	err = initMongo()
	if err != nil {
		return nil, fmt.Errorf("Error initializing mongodb: %s", err.Error())
	}
	return cfg, nil
}

func startDaemon(c *cli.Context) error {
	initLogger(true, true, false)
	cfg, err := prepareConfig(c.String("config"))
	if err != nil {
		logger.Panic(err.Error())
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

func importFiles(c *cli.Context) error {
	initLogger(true, true, false)
	_, err := prepareConfig(c.String("config"))
	if err != nil {
		logger.Panic(err.Error())
	}

	if c.NArg() < 1 {
		logger.Error("At least 1 file should be specified")
		return errors.New("Invalid arguments")
	}

	m := getMongo()
	defer m.Close()

	for _, file := range c.Args() {
		err := importJSON(file, m)
		if err != nil {
			logger.Errorf("Error importing %s: %s", file, err.Error())
		} else {
			logger.Notice("Successfully imported %s", file)
		}
	}
	return nil
}

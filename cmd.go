package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"syscall"

	"github.com/dgiagio/getpass"
	"gopkg.in/mgo.v2/bson"
	"gopkg.in/urfave/cli.v1"
)

func prepareConfig(cfgFile string) *DaemonConfig {
	logger.Noticef("Using config file: %s", cfgFile)
	cfg, err := loadDaemonConfig(cfgFile)
	if err != nil {
		logger.Panic(err.Error())
	}
	err = initMongo()
	if err != nil {
		logger.Panicf(
			fmt.Errorf("Error initializing mongodb: %s", err.Error()).Error())
	}
	return cfg
}

func startDaemon(c *cli.Context) error {
	initLogger(true, c.Bool("debug"), false)
	logger.Notice("Debug mode: %v", c.Bool("debug"))

	cfg := prepareConfig(c.GlobalString("config"))

	ldapListenAddr := fmt.Sprintf("%s:%d", cfg.LDAP.ListenAddr, cfg.LDAP.ListenPort)
	logger.Debugf("Listen LDAP Addr: %s", ldapListenAddr)

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

func changePasswd(c *cli.Context) error {
	if c.NArg() > 1 {
		fmt.Println("You can only change password for one user every time")
		cli.ShowCommandHelp(c, "passwd")
		return errors.New("Invalid arguments")
	}

	initLogger(true, false, false)
	prepareConfig(c.GlobalString("config"))

	curUser, err := user.Current()
	if err != nil {
		err := errors.New("Cannot get current user")
		logger.Error(err.Error())
		return err
	}

	var username string
	if c.NArg() == 1 {
		username = c.Args().Get(0)
	} else {
		username = curUser.Username
	}

	m := getMongo()
	defer m.Close()

	// make sure user exists
	selector := bson.M{"username": username}
	users := m.FindUsers(selector, "")
	if len(users) == 0 {
		err := fmt.Errorf("No such user: %s", username)
		logger.Error(err.Error())
		return err
	}

	// make sure user has permission
	user := users[0]
	if curUser.Uid != "0" && curUser.Uid != strconv.Itoa(user.UID) {
		err := fmt.Errorf("Permission denied (you cannot change other people's password)")
		logger.Error(err.Error())
		return err
	}

	logger.Noticef("Changing password for %s", username)

	// root does not need to type current password
	if curUser.Uid != "0" {
		curPass, _ := getpass.GetPassword("Current Password: ")
		if !user.Authenticate(curPass) {
			err := errors.New("Wrong password")
			logger.Error(err.Error())
			return err
		}
	}

	// get password twice
	newPass, _ := getpass.GetPassword("New Password: ")
	confirmPass, _ := getpass.GetPassword("Confirm Password: ")
	if newPass != confirmPass {
		err := errors.New("Passwords do not match")
		logger.Error(err.Error())
		logger.Notice("Password unchanged")
		return err
	}

	user.Passwd(confirmPass)
	err = m.UserColl().
		Update(selector, bson.M{"$set": bson.M{"password": user.Password}})
	if err != nil {
		logger.Errorf("Failed to update password: %s", err.Error())
		return err
	}

	logger.Notice("Password updated")
	return nil
}

func importFiles(c *cli.Context) error {
	initLogger(true, false, false)
	prepareConfig(c.GlobalString("config"))

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

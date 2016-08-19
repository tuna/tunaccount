package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"strings"
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

func isRootUser() error {
	curUser, err := user.Current()
	if err != nil {
		return errors.New("Cannot get current user")
	}
	if curUser.Uid != "0" {
		return errors.New("Permission denied")
	}
	return nil
}

func cmdNotImplemented(c *cli.Context) error {
	fmt.Println(`This command is not yet implemented (-_-)`)
	return nil
}

// System commands

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

func importFiles(c *cli.Context) error {
	initLogger(true, false, false)
	prepareConfig(c.GlobalString("config"))

	if c.NArg() < 1 {
		logger.Error("At least 1 file should be specified")
		return errors.New("Invalid arguments")
	}

	if err := isRootUser(); err != nil {
		logger.Error(err.Error())
		return err
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

// User Management commands

func cmdUseradd(c *cli.Context) error {
	if c.NArg() != 1 || c.String("email") == "" || c.String("name") == "" {
		fmt.Println("Username, Name and Email are required\n")
		cli.ShowCommandHelp(c, "add")
		return errors.New("Invalid arguments")
	}

	initLogger(true, false, false)

	if err := isRootUser(); err != nil {
		logger.Error(err.Error())
		return err
	}

	cfg := prepareConfig(c.GlobalString("config"))

	m := getMongo()
	defer m.Close()

	user := User{
		UID:        m.getNextSeq("uid"),
		GID:        cfg.TUNA.DefaultGID,
		Username:   c.Args().Get(0),
		Name:       c.String("name"),
		Email:      c.String("email"),
		Phone:      c.String("phone"),
		LoginShell: c.String("shell"),
		IsActive:   true,
	}

	err := m.UserColl().Insert(user)
	if err != nil {
		logger.Errorf("Failed to add user: %s", err.Error())
		return err
	}

	logger.Noticef("Successfully created account: %s", user.Username)
	return nil
}

func cmdPasswd(c *cli.Context) error {
	if c.NArg() > 1 {
		fmt.Println("You can only change password for one user every time\n")
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

// Group Management commands

func cmdGroupList(c *cli.Context) error {
	tag := c.String("tag")

	initLogger(true, false, false)
	if err := isRootUser(); err != nil {
		logger.Error(err.Error())
		return err
	}
	prepareConfig(c.GlobalString("config"))
	m := getMongo()
	defer m.Close()

	groups := []PosixGroup{}
	m.PosixGroupColl().
		Find(bson.M{"is_active": true, "tag": tag}).
		Sort("gid").
		All(&groups)
	for _, group := range groups {
		fmt.Printf(
			"%d:%s: %s\n", group.GID,
			group.Name, strings.Join(group.Members, ","),
		)
	}
	return nil
}

func cmdGroupAdd(c *cli.Context) error {
	if c.NArg() < 1 {
		fmt.Println("Group name is required\n")
		cli.ShowCommandHelp(c, "add")
		return errors.New("Invalid arguments")
	}
	tag := c.String("tag")
	if tag == "" {
		logger.Warning("You are trying to create a universal group, confirm? (yes)")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			logger.Notice("cancelled.")
			return nil
		}
	}

	initLogger(true, false, false)
	if err := isRootUser(); err != nil {
		logger.Error(err.Error())
		return err
	}

	prepareConfig(c.GlobalString("config"))

	m := getMongo()
	defer m.Close()

	groupname := c.Args().Get(0)

	group := PosixGroup{
		GID:      m.getNextSeq("gid"),
		Name:     groupname,
		IsActive: true,
		Tag:      tag,
	}
	err := m.PosixGroupColl().Insert(group)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	logger.Noticef("added group %s", groupname)
	return nil
}

func cmdGroupAddUser(c *cli.Context) error {
	if c.NArg() < 2 {
		fmt.Println("Group name and username is required\n")
		cli.ShowCommandHelp(c, "adduser")
		return errors.New("Invalid arguments")
	}
	tag := c.String("tag")
	if tag == "" {
		logger.Warning("You are trying to add member to a universal group, confirm? (yes)")
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "yes" {
			logger.Notice("cancelled.")
			return nil
		}
	}

	initLogger(true, false, false)
	if err := isRootUser(); err != nil {
		logger.Error(err.Error())
		return err
	}

	prepareConfig(c.GlobalString("config"))

	m := getMongo()
	defer m.Close()

	username := c.Args().Get(0)
	groupname := c.Args().Get(1)

	selector := bson.M{
		"name":      c.Args().Get(1),
		"tag":       tag,
		"is_active": true,
	}

	if cnt, err := m.PosixGroupColl().Find(selector).Count(); err != nil {
		logger.Error(err.Error())
		return err
	} else if cnt != 1 {
		err = fmt.Errorf("Not an exactly-one match for groups: %s [tag: %s]", groupname, tag)
		logger.Error(err.Error())
		return err
	}

	users := m.FindUsers(bson.M{"username": username}, "")
	if len(users) < 1 {
		err := fmt.Errorf("No such user: %s", username)
		logger.Error(err.Error())
		return err
	}

	err := m.PosixGroupColl().
		Update(selector, bson.M{"$addToSet": bson.M{"members": username}})
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	logger.Noticef("user %s added to group %s", username, groupname)
	return nil
}

// Tag Management commands

func cmdTagUser(c *cli.Context) error {
	if c.NArg() < 1 || c.String("tag") == "" {
		fmt.Println("Username and tag are required\n")
		cli.ShowCommandHelp(c, "user")
		return errors.New("Invalid arguments")
	}

	initLogger(true, false, false)
	if err := isRootUser(); err != nil {
		logger.Error(err.Error())
		return err
	}

	prepareConfig(c.GlobalString("config"))

	m := getMongo()
	defer m.Close()

	tag := c.String("tag")
	if err := m.EnsureTag(tag); err != nil {
		logger.Error(err.Error())
		return err
	}

	for _, username := range c.Args() {
		selector := bson.M{"username": username}
		users := m.FindUsers(selector, "")
		if len(users) < 0 {
			logger.Warningf("user %s does not exist", username)
			continue
		}
		m.UserColl().Update(
			selector, bson.M{
				"$addToSet": bson.M{"tags": tag},
			},
		)
	}

	return nil

}

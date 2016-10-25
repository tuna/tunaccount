package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/urfave/cli.v1"
)

var (
	buildstamp = ""
	githash    = "no githash provided"
)

func main() {

	app := cli.NewApp()
	app.Name = "tunaccount"
	app.EnableBashCompletion = true
	cli.VersionPrinter = func(c *cli.Context) {
		var builddate string
		if buildstamp == "" {
			builddate = "No build data provided"
		} else {
			ts, err := strconv.Atoi(buildstamp)
			if err != nil {
				builddate = "No build data provided"
			} else {
				t := time.Unix(int64(ts), 0)
				builddate = t.String()
			}
		}
		fmt.Printf(
			"Version: %s\n"+
				"Git Hash: %s\n"+
				"Build Data: %s\n",
			c.App.Version, githash, builddate,
		)
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config, c",
			Value:  "/etc/tunaccountd.conf",
			Usage:  "specify configuration file",
			EnvVar: "TUNACCOUNT_CONFIG_FILE",
		},
		cli.StringFlag{
			Name:   "server-url",
			Value:  "http://127.0.0.1:9501",
			Usage:  "tunaccount HTTP base-url",
			EnvVar: "TUNACCOUNT_SERVER_URL",
		},
		cli.BoolFlag{
			Name:   "debug",
			Usage:  "enable debug",
			EnvVar: "TUNACCOUNT_DEBUG",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:   "daemon",
			Usage:  "run tunaccount daemon",
			Action: startDaemon,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "debug",
					Usage: "enable debug",
				},
				cli.StringFlag{
					Name:  "root-password",
					Usage: "Temporary root password, only used for initilization",
				},
			},
		},
		{
			Name:      "import",
			Usage:     "import json files to tunaccount",
			Action:    importFiles,
			ArgsUsage: "[files...]",
		},
		{
			Name:  "user",
			Usage: "user management",
			Subcommands: []cli.Command{
				{
					Name:    "list",
					Usage:   "list users",
					Aliases: []string{"ls"},
					Action:  cmdNotImplemented,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "tag, t",
							Usage: "tag",
						},
					},
				},
				{
					Name:      "add",
					Usage:     "add a user",
					Action:    cmdUseradd,
					ArgsUsage: "<username>",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "shell, s",
							Usage: "Login shell of the new account",
							Value: "/bin/bash",
						},
						cli.StringFlag{
							Name:  "name",
							Usage: "Fullname of the new account (Required)",
						},
						cli.StringFlag{
							Name:  "email, mail",
							Usage: "Email address of the new account (Required)",
						},
						cli.StringFlag{
							Name:  "phone, mobile",
							Usage: "Phone number of the new account",
						},
					},
				},
				{
					Name:      "passwd",
					Usage:     "set password of a user, default is current user",
					Action:    cmdPasswd,
					ArgsUsage: "[username]",
				},
				{
					Name:      "modify",
					Usage:     "modify user infomation",
					Aliases:   []string{"mod"},
					Action:    cmdNotImplemented,
					ArgsUsage: "<username>",
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "shell, s",
							Usage: "Login shell of the new account",
							Value: "/bin/bash",
						},
						cli.StringFlag{
							Name:  "name",
							Usage: "Fullname of the new account (Required)",
						},
						cli.StringFlag{
							Name:  "email, mail",
							Usage: "Email address of the new account (Required)",
						},
						cli.StringFlag{
							Name:  "phone, mobile",
							Usage: "Phone number of the new account",
						},
					},
				},
				{
					Name:      "del",
					Usage:     "delete a user",
					Action:    cmdNotImplemented,
					ArgsUsage: "<username>",
				},
			},
		},
		{
			Name:  "group",
			Usage: "group management",
			Subcommands: []cli.Command{
				{
					Name:    "list",
					Aliases: []string{"ls"},
					Usage:   "list groups",
					Action:  cmdGroupList,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "tag, t",
							Usage: "tag",
						},
					},
				},
				{
					Name:    "new",
					Aliases: []string{"add"},
					Usage:   "create a group",
					Action:  cmdGroupAdd,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "tag, t",
							Usage: "group tag",
						},
					},
				},
				{
					Name:      "adduser",
					Usage:     "add a user to a group",
					ArgsUsage: "<username> <groupname>",
					Action:    cmdGroupAddUser,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "tag, t",
							Usage: "group tag",
						},
					},
				},
				{
					Name:      "deluser",
					Usage:     "del users from a group",
					ArgsUsage: "<username> <groupname>",
					Action:    cmdNotImplemented,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "tag, t",
							Usage: "group tag",
						},
					},
				},
			},
		},
		{
			Name:  "tag",
			Usage: "tag management",
			Subcommands: []cli.Command{
				{
					Name:    "new",
					Aliases: []string{"add"},
					Usage:   "add new tag",
					Action:  cmdNotImplemented,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "desc, d",
							Usage: "tag description",
						},
					},
				},
				{
					Name:    "list",
					Aliases: []string{"ls"},
					Action:  cmdNotImplemented,
					Usage:   "list tags",
				},
				{
					Name:      "user",
					Usage:     "tag users",
					ArgsUsage: "<users>",
					Action:    cmdTagUser,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "tag, t",
							Usage: "tag name (Required)",
						},
					},
				},
			},
		},
	}

	app.Run(os.Args)
}

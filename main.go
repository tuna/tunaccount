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

	app.Commands = []cli.Command{
		{
			Name:   "daemon",
			Usage:  "run tunaccount daemon",
			Action: startDaemon,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "config, c",
					Value: "/etc/tunaccount.conf",
					Usage: "specify configuration file",
				},
			},
		},
	}

	app.Run(os.Args)
}

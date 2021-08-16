package main

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
)



var App = cli.NewApp()

func main() {
	if err := App.Run(os.Args); err != nil {
		logrus.WithField("error", err).Info("App Exist with Error")
	}
}

package main

import (
	"os"
	"os/signal"

	"github.com/drkaka/lg"
	"github.com/goinout/goinout"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "goinout"
	app.Usage = "goinout application"
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug",
			Usage: "starting with debug model",
		},
		cli.StringFlag{
			Name:  "inputs, i",
			Usage: "inputs folder path",
			Value: "./inputs",
		},
		cli.StringFlag{
			Name:  "outputs, o",
			Usage: "outputs folder path",
			Value: "./outputs",
		},
	}

	app.Action = func(c *cli.Context) error {
		// set the input folder
		inputPath := c.GlobalString("inputs")
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			os.Mkdir(inputPath, os.ModePerm)
		}

		// set the output folder
		outputPath := c.GlobalString("outputs")
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			os.Mkdir(outputPath, os.ModePerm)
		}

		goinout.Start(inputPath, outputPath)

		// wait for os.Interrupt to quit
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt)

		select {
		case <-sigs:
		}

		return nil
	}

	app.Before = func(c *cli.Context) error {
		if c.GlobalBool("debug") {
			lg.InitLogger(true)
		} else {
			lg.InitLogger(false)
		}
		lg.L(nil).Debug("debug model")
		return nil
	}

	app.After = func(c *cli.Context) error {
		goinout.Stop()
		return nil
	}

	app.Run(os.Args)
}

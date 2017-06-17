package main

import (
	"os"

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
			Name:  "input, i",
			Usage: "input folder path",
			Value: "./input",
		},
		cli.StringFlag{
			Name:  "output, o",
			Usage: "output folder path",
			Value: "./output",
		},
	}

	app.Action = func(c *cli.Context) error {
		// set the input folder
		inputPath := c.GlobalString("input")
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			os.Mkdir(inputPath, os.ModePerm)
		}

		// set the output folder
		outputPath := c.GlobalString("output")
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			os.Mkdir(outputPath, os.ModePerm)
		}

		goinout.Start(inputPath, outputPath)
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

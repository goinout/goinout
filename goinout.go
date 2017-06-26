package goinout

import (
	"path/filepath"
	"plugin"
	"strings"

	"github.com/drkaka/lg"
	"github.com/rjeczalik/notify"
	"go.uber.org/zap"
)

type loadFunc func(*plugin.Plugin, string)

// Start the service
func Start(input, output string) {
	inputsPath = input
	outputsPath = output

	getOutputPlugins()
	startInputPlugins()

	observeInputs()
	observeOutputs()
	lg.L(nil).Info("started", zap.String("inputsPath", input), zap.String("outputsPath", output))
}

// Stop and cleanup
func Stop() {
	lg.L(nil).Info("stopped")
}

func extOK(file string) bool {
	return filepath.Ext(file) == ".so"
}

func pluginName(file string) string {
	return strings.TrimSuffix(file, filepath.Ext(file))
}

func loadPlugin(file string, load loadFunc) {
	if !extOK(file) {
		lg.L(nil).Debug("invalid input ext", zap.String("file", file))
		return
	}

	p, err := plugin.Open(file)
	if err != nil {
		lg.L(nil).Error("open file wrong", zap.Error(err))
		return
	}

	load(p, pluginName(filepath.Base(file)))
}

func observe(folderPath string, load loadFunc, delete func(string), rename func(string, string)) {
	c := make(chan notify.EventInfo, 1)

	if err := notify.Watch(folderPath, c, notify.InCloseWrite, notify.InDelete, notify.InMovedTo, notify.InMovedFrom); err != nil {
		lg.L(nil).Panic("watcher wrong", zap.Error(err))
	}
	defer notify.Stop(c)

	// Process events
	go func() {
		for {
			ei := <-c
			fileName := filepath.Base(ei.Path())
			name := pluginName(fileName)

			switch ei.Event() {
			case notify.InDelete:
				delete(name)
			case notify.InCloseWrite:
				loadPlugin(ei.Path(), load)
			case notify.InMovedTo:
				// Rename event should have a follow-up InMovedFrom event
				fromEV := <-c
				if fromEV.Event() != notify.InMovedFrom {
					lg.L(nil).Error("Expecting create event", zap.Any("event", fromEV))
				} else {
					from := pluginName(filepath.Base(fromEV.Path()))
					if extOK(from) {
						rename(from, name)
					}
				}
			case notify.InMovedFrom:
				// actually a delete event, because maybe move to Trash
				delete(name)
			}
		}
	}()
}

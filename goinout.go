package goinout

import (
	"path/filepath"
	"strings"

	"github.com/drkaka/lg"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

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

func observe(folderPath string, add, reload, delete func(string), rename func(string, string)) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	// Process events
	go func() {
		for {
			select {
			case ev := <-watcher.Events:
				fileName := filepath.Base(ev.Name)
				if !extOK(fileName) {
					continue
				}
				if ev.Op&fsnotify.Create == fsnotify.Create {
					add(fileName)
				} else if ev.Op&fsnotify.Write == fsnotify.Write {
					reload(fileName)
				} else if ev.Op&fsnotify.Rename == fsnotify.Rename {
					from := fileName
					// Rename event should have a follow-up create event
					addEV := <-watcher.Events
					if addEV.Op&fsnotify.Create != fsnotify.Create {
						lg.L(nil).Error("Expecting create event", zap.Any("event", addEV))
					} else {
						to := filepath.Base(addEV.Name)
						if extOK(to) {
							rename(from, to)
						}
					}
				} else if ev.Op&fsnotify.Remove == fsnotify.Remove {
					delete(fileName)
				}
			case err := <-watcher.Errors:
				lg.L(nil).Error("error", zap.Error(err))
			}
		}
	}()

	if err := watcher.Add(folderPath); err != nil {
		panic(err)
	}
}

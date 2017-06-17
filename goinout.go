package goinout

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/drkaka/lg"
	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

type outputFunc func([]byte) error

type inputStruct struct {
	all map[string]context.Context
	l   *sync.RWMutex
}

type outputStruct struct {
	all map[string]outputFunc
	l   *sync.RWMutex
}

var (
	inputsPath  string
	outputsPath string

	inputs  *inputStruct
	outputs *outputStruct
)

func init() {
	inputs = &inputStruct{
		all: make(map[string]context.Context, 0),
		l:   new(sync.RWMutex),
	}

	outputs = &outputStruct{
		all: make(map[string]outputFunc, 0),
		l:   new(sync.RWMutex),
	}
}

// Start the service
func Start(input, output string) {
	inputsPath = input
	outputsPath = output

	loadOutputs()
	loadInputs()

	observeInputs()
	observeOutputs()
	lg.L(nil).Info("started", zap.String("inputsPath", input), zap.String("outputsPath", output))
}

// Stop and cleanup
func Stop() {
	lg.L(nil).Info("stopped")
}

func extOK(file string) bool {
	return filepath.Ext(file) == "so"
}

func loadInputs() {
	files, err := ioutil.ReadDir(inputsPath)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		fmt.Println(file.Name())
	}
}

func loadOutputs() {
	files, err := ioutil.ReadDir(outputsPath)
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		fileName := file.Name()
		if !extOK(fileName) {
			continue
		}
		fn, err := loadOutput(fileName)
		if err != nil {
			lg.L(nil).Warn("load output error", zap.Error(err))
		} else {
			outputs.l.Lock()
			outputs.all[fileName] = fn
			outputs.l.Unlock()
		}
	}
}

func addInput(plugin string) {
	lg.L(nil).Debug("input plugin add", zap.String("plugin", plugin))

}

func reloadInput(plugin string) {
	lg.L(nil).Debug("input plugin reload", zap.String("plugin", plugin))
}

func renameInput(from, to string) {
	lg.L(nil).Debug("input plugin rename", zap.String("from", from), zap.String("to", to))
}

func deleteInput(plugin string) {
	lg.L(nil).Debug("input plugin delete", zap.String("plugin", plugin))
}

func addOutput(plugin string) {
	lg.L(nil).Debug("Output plugin add", zap.String("plugin", plugin))
}

func reloadOutput(plugin string) {
	lg.L(nil).Debug("Output plugin reload", zap.String("plugin", plugin))
}

func renameOutput(from, to string) {
	lg.L(nil).Debug("Output plugin rename", zap.String("from", from), zap.String("to", to))
}

func deleteOutput(plugin string) {
	lg.L(nil).Debug("Output plugin delete", zap.String("plugin", plugin))
}

func observeInputs() {
	observe(inputsPath, addInput, reloadInput, deleteInput, renameInput)
}

func observeOutputs() {
	observe(outputsPath, addOutput, reloadOutput, deleteOutput, renameOutput)
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
				if ev.Op&fsnotify.Create == fsnotify.Create {
					add(fileName)
				} else if ev.Op&fsnotify.Write == fsnotify.Write {
					reload(fileName)
				} else if ev.Op&fsnotify.Rename == fsnotify.Rename {
					from := fileName
					addEV := <-watcher.Events
					if addEV.Op&fsnotify.Create != fsnotify.Create {
						lg.L(nil).Error("Expecting create event", zap.Any("event", addEV))
					} else {
						to := filepath.Base(addEV.Name)
						rename(from, to)
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

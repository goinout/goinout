package goinout

import (
	"context"
	"sync"

	"github.com/drkaka/lg"
	"github.com/howeyc/fsnotify"
	"go.uber.org/zap"
)

type inputStruct struct {
	all map[string]context.Context
	l   *sync.RWMutex
}

type outputStruct struct {
	all map[string]func([]byte) error
	l   *sync.RWMutex
}

var (
	inputPath  string
	outputPath string

	inputs  *inputStruct
	outputs *outputStruct
)

func init() {
	inputs = &inputStruct{
		all: make(map[string]context.Context, 0),
		l:   new(sync.RWMutex),
	}

	outputs = &outputStruct{
		all: make(map[string]func([]byte) error, 0),
		l:   new(sync.RWMutex),
	}
}

// Start the service
func Start(input, output string) {
	inputPath = input
	outputPath = output

	lg.L(nil).Info("started", zap.String("inputPath", input), zap.String("outputPath", output))
}

// Stop and cleanup
func Stop() {
	lg.L(nil).Info("stopped")
}

func addInput(plugin string) {

}

func reloadInput(plugin string) {

}

func renameInput(plugin string) {

}

func deleteInput(plugin string) {

}

func addOutput(plugin string) {

}

func reloadOutput(plugin string) {

}

func renameOutput(plugin string) {

}

func deleteOutput(plugin string) {

}

func observeInput() {

}

func observeOutput() {

}

func observe(path string, add, reload, rename, delete func(string)) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	// Process events
	go func() {
		for {
			select {
			case ev := <-watcher.Event:
				lg.L(nil).Debug("event", zap.Any("event", ev))
				fileName := ev.Name
				if ev.IsCreate() {
					add(fileName)
				} else if ev.IsModify() {
					reload(fileName)
				} else if ev.IsRename() {
					rename(fileName)
				} else if ev.IsDelete() {
					delete(fileName)
				}
			case err := <-watcher.Error:
				lg.L(nil).Error("error", zap.Error(err))
			}
		}
	}()

	if err := watcher.Watch(inputPath); err != nil {
		panic(err)
	}
}

package goinout

import (
	"context"
	"errors"
	"io/ioutil"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/drkaka/lg"
	"go.uber.org/zap"
)

type inputStruct struct {
	all map[string]context.CancelFunc
	l   *sync.RWMutex
}

// InputFunc type
type InputFunc func(context.Context, OutputGetter)

var (
	//ErrInputTypeWrong error
	ErrInputTypeWrong = errors.New("input function type wrong")

	inputsPath string
	inputs     *inputStruct
)

func init() {
	inputs = &inputStruct{
		all: make(map[string]context.CancelFunc, 0),
		l:   new(sync.RWMutex),
	}
}

func observeInputs() {
	observe(inputsPath, addInput, reloadInput, deleteInput, renameInput)
}

func startInputPlugins() {
	files, err := ioutil.ReadDir(inputsPath)
	if err != nil {
		panic(err)
	}
	lg.L(nil).Debug("load inputs", zap.Int("count", len(files)))

	for _, file := range files {
		loadInput(file.Name())
	}
}

// loadInput plugin and start
func loadInput(fileName string) {
	if !extOK(fileName) {
		lg.L(nil).Debug("invalid input ext", zap.String("file", fileName))
		return
	}

	f := filepath.Join(inputsPath, fileName)
	p, err := plugin.Open(f)
	if err != nil {
		lg.L(nil).Warn("input plugin open failed", zap.Error(err))
		return
	}

	fn, err := p.Lookup("Start")
	if err != nil {
		lg.L(nil).Warn("input plugin lookup failed", zap.Error(err))
		return
	}

	inputFn, ok := fn.(func(context.Context, OutputGetter))
	if !ok {
		lg.L(nil).Warn(ErrInputTypeWrong.Error())
		return
	}

	// start the input
	inputContext, cancelFunc := context.WithCancel(context.Background())
	go inputFn(inputContext, getOutputFunc)

	inputs.l.Lock()
	inputs.all[pluginName(fileName)] = cancelFunc
	inputs.l.Unlock()

	lg.L(nil).Debug("input plugin successfully loaded and started", zap.String("plugin", fileName))
}

func addInput(plugin string) {
	lg.L(nil).Debug("input plugin add event", zap.String("plugin", plugin))

	inputs.l.RLock()
	cancel, ok := inputs.all[pluginName(plugin)]
	inputs.l.RUnlock()

	// if name existed, cancel it and load
	if ok {
		lg.L(nil).Warn("already existed")
		cancel()
	}
	loadInput(plugin)
}

func reloadInput(plugin string) {
	lg.L(nil).Debug("input plugin reload event", zap.String("plugin", plugin))

	inputs.l.RLock()
	cancel, ok := inputs.all[pluginName(plugin)]
	inputs.l.RUnlock()

	// if name existed, cancel it and load
	if ok {
		cancel()
	} else {
		lg.L(nil).Warn("not existed")
	}
	loadInput(plugin)
}

func renameInput(from, to string) {
	lg.L(nil).Debug("input plugin rename event", zap.String("from", from), zap.String("to", to))

	inputs.l.RLock()
	fn, ok := inputs.all[pluginName(from)]
	inputs.l.RUnlock()

	if !ok {
		lg.L(nil).Warn("can't rename missing old input plugin name")
		return
	}

	// delete the old key and assign to the new key
	inputs.l.Lock()
	delete(inputs.all, pluginName(from))
	inputs.all[pluginName(to)] = fn
	inputs.l.Unlock()
}

func deleteInput(plugin string) {
	lg.L(nil).Debug("input plugin delete event", zap.String("plugin", plugin))

	inputs.l.RLock()
	cancel, ok := inputs.all[pluginName(plugin)]
	inputs.l.RUnlock()

	if ok {
		// stop and delete the record
		cancel()

		inputs.l.Lock()
		delete(inputs.all, pluginName(plugin))
		inputs.l.Unlock()
	} else {
		lg.L(nil).Warn("missing input plugin name")
	}
}

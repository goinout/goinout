package goinout

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"plugin"
	"reflect"
	"sync"

	"github.com/drkaka/lg"
	"go.uber.org/zap"
)

type outputStruct struct {
	all map[string]OutputFunc
	l   *sync.RWMutex
}

// OutputGetter type
type OutputGetter func(string) OutputFunc

// OutputFunc type
type OutputFunc func(map[string]interface{}) error

var (
	//ErrOutputTypeWrong error
	ErrOutputTypeWrong = errors.New("output function type wrong")

	outputsPath string
	outputs     *outputStruct
)

func init() {
	outputs = &outputStruct{
		all: make(map[string]OutputFunc, 0),
		l:   new(sync.RWMutex),
	}
}

func observeOutputs() {
	observe(outputsPath, loadOutput, deleteOutput, renameOutput)
}

func getOutputPlugins() {
	files, err := ioutil.ReadDir(outputsPath)
	if err != nil {
		panic(err)
	}
	lg.L(nil).Debug("load outputs", zap.Int("found", len(files)))
	for _, file := range files {
		loadPlugin(filepath.Join(outputsPath, file.Name()), loadOutput)
	}

	outputs.l.RLock()
	var loaded []string
	for k := range outputs.all {
		loaded = append(loaded, k)
	}
	lg.L(nil).Info("outputs loaded", zap.Strings("plugins", loaded))
	outputs.l.RUnlock()
}

func getOutputFunc(name string) func(map[string]interface{}) error {
	outputs.l.RLock()
	fn := outputs.all[name]
	outputs.l.RUnlock()

	lg.L(nil).Debug("get output", zap.Any("plugin", reflect.TypeOf(fn)))
	return fn
}

func loadOutput(plug *plugin.Plugin, name string) {
	fn, err := plug.Lookup("Handle")
	if err != nil {
		lg.L(nil).Warn("output plugin lookup failed", zap.Error(err))
		return
	}

	outputFn, ok := fn.(func(map[string]interface{}) error)
	if !ok {
		lg.L(nil).Debug("wrong ouput type", zap.Any("type", reflect.TypeOf(outputFn)))
		lg.L(nil).Warn(ErrOutputTypeWrong.Error())
		return
	}

	outputs.l.Lock()
	outputs.all[name] = outputFn
	outputs.l.Unlock()

	lg.L(nil).Debug("output plugin successfully loaded and added", zap.String("plugin", name))
}

func renameOutput(from, to string) {
	lg.L(nil).Debug("Output plugin rename event", zap.String("from", from), zap.String("to", to))

	outputs.l.RLock()
	fn, ok := outputs.all[from]
	outputs.l.RUnlock()

	if !ok {
		lg.L(nil).Warn("can't rename missing old output plugin name")
		return
	}

	outputs.l.Lock()
	delete(outputs.all, from)
	outputs.all[to] = fn
	outputs.l.Unlock()
}

func deleteOutput(name string) {
	lg.L(nil).Debug("Output plugin delete event", zap.String("plugin", name))

	outputs.l.Lock()
	delete(outputs.all, name)
	outputs.l.Unlock()
}

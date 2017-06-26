package goinout

import (
	"errors"
	"io/ioutil"
	"path/filepath"
	"plugin"
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
type OutputFunc func(map[string]interface{}, []byte) error

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
	observe(outputsPath, addOutput, reloadOutput, deleteOutput, renameOutput)
}

func getOutputPlugins() {
	files, err := ioutil.ReadDir(outputsPath)
	if err != nil {
		panic(err)
	}
	lg.L(nil).Debug("load outputs", zap.Int("count", len(files)))
	for _, file := range files {
		addOutput(file.Name())
	}
}

func loadOutput(fileName string) {
	if !extOK(fileName) {
		lg.L(nil).Debug("invalid output ext", zap.String("file", fileName))
		return
	}

	f := filepath.Join(outputsPath, fileName)
	p, err := plugin.Open(f)
	if err != nil {
		lg.L(nil).Warn("output plugin open failed", zap.Error(err))
		return
	}

	fn, err := p.Lookup("Handle")
	if err != nil {
		lg.L(nil).Warn("output plugin lookup failed", zap.Error(err))
		return
	}

	outputFn, ok := fn.(func(map[string]interface{}, []byte) error)
	if !ok {
		lg.L(nil).Warn(ErrOutputTypeWrong.Error())
		return
	}

	outputs.l.Lock()
	outputs.all[pluginName(fileName)] = outputFn
	outputs.l.Unlock()

	lg.L(nil).Debug("output plugin successfully loaded and added", zap.String("plugin", fileName))
}

func getOutputFunc(name string) OutputFunc {
	outputs.l.RLock()
	fn := outputs.all[name]
	outputs.l.RUnlock()

	return fn
}

func addOutput(plugin string) {
	lg.L(nil).Debug("Output plugin add event", zap.String("plugin", plugin))
	loadOutput(plugin)
}

func reloadOutput(plugin string) {
	lg.L(nil).Debug("Output plugin reload event", zap.String("plugin", plugin))
	loadOutput(plugin)
}

func renameOutput(from, to string) {
	lg.L(nil).Debug("Output plugin rename event", zap.String("from", from), zap.String("to", to))

	outputs.l.RLock()
	fn, ok := outputs.all[pluginName(from)]
	outputs.l.RUnlock()

	if !ok {
		lg.L(nil).Warn("can't rename missing old output plugin name")
		return
	}

	outputs.l.Lock()
	delete(outputs.all, pluginName(from))
	outputs.all[pluginName(to)] = fn
	outputs.l.Unlock()
}

func deleteOutput(plugin string) {
	lg.L(nil).Debug("Output plugin delete event", zap.String("plugin", plugin))

	outputs.l.Lock()
	delete(outputs.all, pluginName(plugin))
	outputs.l.Unlock()
}

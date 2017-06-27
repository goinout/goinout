package goinout

import (
	"context"
	"errors"
	"io/ioutil"
	"path/filepath"
	"plugin"
	"reflect"
	"sync"

	"github.com/drkaka/lg"
	"go.uber.org/zap"
)

type inputRecord struct {
	cancelFn context.CancelFunc
	ctx      context.Context
}

type inputStruct struct {
	all map[string]*inputRecord
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
		all: make(map[string]*inputRecord, 0),
		l:   new(sync.RWMutex),
	}
}

func observeInputs() {
	observe(inputsPath, loadInput, deleteInput, renameInput)
}

func startInputPlugins() {
	files, err := ioutil.ReadDir(inputsPath)
	if err != nil {
		panic(err)
	}
	lg.L(nil).Debug("load inputs", zap.Int("found", len(files)))

	for _, file := range files {
		loadPlugin(filepath.Join(inputsPath, file.Name()), loadInput)
	}

	var loaded []string
	for k := range inputs.all {
		loaded = append(loaded, k)
	}
	lg.L(nil).Info("inputs loaded", zap.Strings("plugins", loaded))
}

// loadInput plugin and start
//
// name is string without ext
func loadInput(plug *plugin.Plugin, name string) {
	// Extract the Exchange function
	fn, err := plug.Lookup("Exchange")
	if err != nil {
		lg.L(nil).Warn("input plugin Exchange func not found", zap.Error(err))
		return
	}

	exchangeCtx, ok := fn.(func(context.Context) context.Context)
	if !ok {
		lg.L(nil).Debug("wrong input Exchange type", zap.Any("type", reflect.TypeOf(fn)))
		lg.L(nil).Warn(ErrInputTypeWrong.Error())
		return
	}

	// Extract the Start function
	fn, err = plug.Lookup("Start")
	if err != nil {
		lg.L(nil).Warn("input plugin Start func not found", zap.Error(err))
		return
	}

	inputFn, ok := fn.(func(func(string) func(map[string]interface{}) error))
	if !ok {
		lg.L(nil).Debug("wrong input Start type", zap.Any("type", reflect.TypeOf(inputFn)))
		lg.L(nil).Warn(ErrInputTypeWrong.Error())
		return
	}

	// stop the old if exist
	var old *inputRecord

	inputs.l.RLock()
	old = inputs.all[name]
	inputs.l.RUnlock()

	if old != nil {
		old.cancelFn()
		if old.ctx != nil {
			// wait until done
			<-old.ctx.Done()
		}
	}
	lg.L(nil).Debug("old input stopped", zap.String("plugin", name))

	// create the new one
	inputContext, cancelFunc := context.WithCancel(context.Background())
	stopContext := exchangeCtx(inputContext)

	inputs.l.Lock()
	inputs.all[name] = &inputRecord{
		cancelFn: cancelFunc,
		ctx:      stopContext,
	}
	inputs.l.Unlock()

	// start the input
	go inputFn(getOutputFunc)

	lg.L(nil).Debug("input plugin successfully loaded and started", zap.String("plugin", name))
}

func renameInput(from, to string) {
	lg.L(nil).Debug("input plugin rename event", zap.String("from", from), zap.String("to", to))

	inputs.l.RLock()
	fn, ok := inputs.all[from]
	inputs.l.RUnlock()

	if !ok {
		lg.L(nil).Warn("can't rename missing old input plugin name")
		return
	}

	// delete the old key and assign to the new key
	inputs.l.Lock()
	delete(inputs.all, from)
	inputs.all[to] = fn
	inputs.l.Unlock()
}

func deleteInput(name string) {
	lg.L(nil).Debug("input plugin delete event", zap.String("plugin", name))

	inputs.l.RLock()
	record, ok := inputs.all[name]
	inputs.l.RUnlock()

	if ok {
		// stop and delete the record
		record.cancelFn()

		inputs.l.Lock()
		delete(inputs.all, name)
		inputs.l.Unlock()
	} else {
		lg.L(nil).Warn("missing input plugin name")
	}
}

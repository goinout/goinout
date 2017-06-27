package goinout

import (
	"path/filepath"
	"plugin"
	"strings"
	"sync"

	"time"

	"github.com/drkaka/lg"
	"github.com/rjeczalik/notify"
	"go.uber.org/zap"
)

type loadFunc func(*plugin.Plugin, string)

// move event
type move struct {
	from string
	to   string
}

type moves struct {
	all map[uint32]move
	l   *sync.RWMutex
}

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

	// Process events
	go func() {
		moving := moves{
			all: make(map[uint32]move),
			l:   new(sync.RWMutex),
		}
		for {
			ei := <-c
			fileName := filepath.Base(ei.Path())
			name := pluginName(fileName)
			lg.L(nil).Debug("file changed", zap.String("path", ei.Path()), zap.String("event", ei.Event().String()))

			switch ei.Event() {
			case notify.InDelete:
				delete(name)
			case notify.InCloseWrite:
				loadPlugin(ei.Path(), load)
			case notify.InMovedTo:
				cookie := ei.Sys().(*unix.InotifyEvent).Cookie
				if cookie == uint32(0) {
					lg.L(nil).Error("cookie is 0", zap.Any("event", ei))
				} else {
					// set to value
					moving.l.Lock()
					info := moving[cookie]
					info.to = ei.Path()
					moving[cookie] = info
					moving.l.Unlock()

					handleMoveing(&moving, cookie, load, delete, rename)
				}
			case notify.InMovedFrom:
				cookie := ei.Sys().(*unix.InotifyEvent).Cookie
				if cookie == uint32(0) {
					lg.L(nil).Error("cookie is 0", zap.Any("event", ei))
				} else {
					// set from value
					moving.l.Lock()
					info := moving[cookie]
					info.from = ei.Path()
					moving[cookie] = info
					moving.l.Unlock()

					handleMoveing(&moving, cookie, load, delete, rename)
				}
			}
		}
	}()
}

func handleMoveing(m *moves, ck uint32, load loadFunc, del func(string), ren func(string, string)) {
	m.l.RLock()
	info := m[ck]
	m.l.RUnlock()

	if info.from != "" && info.to != "" {
		// rename event generated
		rename(filepath.Base(pluginName(info.from)), filepath.Base(pluginName(info.to)))

		// delete this record
		m.l.Lock()
		delete(m, ck)
		m.l.Unlock()
	} else {
		go func() {
			// final check after 0.5s, because move job can be done between different folders
			<-time.After(500 * time.Millisecond)
			finalCheck(m, ck, load, del, ren)
		}()
	}
}

func finalCheck(m *moves, ck uint32, load loadFunc, del func(string), ren func(string, string)) {
	m.l.RLock()
	info, ok := m[cookie]
	m.l.RUnlock()

	// the info still existed
	if ok {
		if info.to != "" {
			// to not empty means file moved to this place
			loadPlugin(info.to, load)
		} else if info.from != "" {
			// from not empty means the file moved away
			del(filepath.Base(pluginName(info.from)))
		}

		// delete this record
		m.l.Lock()
		delete(m, ck)
		m.l.Unlock()
	}
}

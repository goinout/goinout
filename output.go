package goinout

import (
	"errors"
	"path/filepath"
	"plugin"

	"github.com/drkaka/lg"
)

// OutputFunc type
type OutputFunc func(map[string]interface{}, []byte) error

var (
	//ErrOutputTypeWrong error
	ErrOutputTypeWrong = errors.New("output function type wrong")
)

func loadOutput(fileName string) (OutputFunc, error) {
	f := filepath.Join(outputsPath, fileName)
	p, err := plugin.Open(f)
	if err != nil {
		lg.L(nil).Debug("plugin open failed")
		return nil, err
	}

	fn, err := p.Lookup("Handle")
	if err != nil {
		return nil, err
	}

	outputFn, ok := fn.(func(map[string]interface{}, []byte) error)
	if !ok {
		return nil, ErrOutputTypeWrong
	}
	return outputFn, nil
}

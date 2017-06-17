package goinout

import (
	"errors"
	"path/filepath"
	"plugin"
)

var (
	//ErrOutputTypeWrong error
	ErrOutputTypeWrong = errors.New("output function type wrong")
)

func loadOutput(fileName string) (outputFunc, error) {
	f := filepath.Join(outputsPath, fileName)
	p, err := plugin.Open(f)
	if err != nil {
		return nil, err
	}

	fn, err := p.Lookup("Handle")
	if err != nil {
		return nil, err
	}

	outputFn, ok := fn.(outputFunc)
	if !ok {
		return nil, ErrOutputTypeWrong
	}
	return outputFn, nil
}

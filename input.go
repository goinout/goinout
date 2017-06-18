package goinout

import (
	"context"
	"errors"
	"path/filepath"
	"plugin"

	"github.com/drkaka/lg"
)

var (
	//ErrInputTypeWrong error
	ErrInputTypeWrong = errors.New("input function type wrong")
)

func loadIntput(fileName string) (context.CancelFunc, error) {
	f := filepath.Join(inputsPath, fileName)
	p, err := plugin.Open(f)
	if err != nil {
		lg.L(nil).Debug("plugin open failed")
		return nil, err
	}

	fn, err := p.Lookup("Start")
	if err != nil {
		return nil, err
	}

	inputFn, ok := fn.(func(context.Context))
	if !ok {
		return nil, ErrInputTypeWrong
	}

	// start the input
	inputContext, cancelFunc := context.WithCancel(ctx)
	inputFn(inputContext)

	return cancelFunc, nil
}

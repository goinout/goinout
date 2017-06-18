package goinout

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtOK(t *testing.T) {
	fileName := "file.so"
	assert.True(t, extOK(fileName), "extension wrong")
}

func TestPluginName(t *testing.T) {
	fileName := "file.so"
	assert.Equal(t, "file", pluginName(fileName), "get plugin name wrong")
}

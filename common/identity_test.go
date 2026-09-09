package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskUsername(t *testing.T) {
	tests := map[string]string{
		"":         "***",
		"A":        "*",
		"AB":       "A*",
		"Alice":    "A***e",
		"张三":       "张*",
		"杨宇彬":      "杨*彬",
		"  user  ": "u**r",
	}
	for input, expected := range tests {
		assert.Equal(t, expected, MaskUsername(input), input)
	}
}

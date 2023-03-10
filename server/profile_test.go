package main_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/svenssonaxel/mattermost-plugin-character-profiles/server"
)

func TestEncodeDecode(t *testing.T) {
	p := main.Plugin{}
	p1 := main.Profile{Name: "Name "}
	p2, err := p.DecodeProfileFromByte(p.EncodeToByte(&p1))
	assert.Nil(t, err)
	assert.NotNil(t, p2)
	assert.Equal(t, p1, *p2)
}

func TestDecode(t *testing.T) {
	p := main.Plugin{}
	profile, err := p.DecodeProfileFromByte([]byte{})
	assert.NotNil(t, err)
	assert.Nil(t, profile)
}

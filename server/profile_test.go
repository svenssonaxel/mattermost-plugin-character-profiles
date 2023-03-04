package main_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/svenssonaxel/mattermost-plugin-pc-npc/server"
)

func TestEncodeDecode(t *testing.T) {
	p := main.Plugin{}
	p1 := main.Profile{Name: "Name "}
	p2 := p.DecodeProfileFromByte(p.EncodeToByte(p1))
	assert.NotNil(t, p2)
	assert.Equal(t, p1, *p2)
}

func TestDecode(t *testing.T) {
	p := main.Plugin{}
	profile := p.DecodeProfileFromByte([]byte{})
	assert.Nil(t, profile)
}

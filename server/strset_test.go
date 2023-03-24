package main_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/svenssonaxel/mattermost-plugin-character-profiles/server"
)

func TestStrset(t *testing.T) {
	// Initialize the backend mock
	be := main.BackendMock{
		IdCounter: new(int),
		KVStore:   map[string][]byte{},
	}
	msg := "TestStrset"
	// Push some values to various lists
	var (
		err    *model.AppError
		strset []string
		l1     = "test1"
		l2     = "test2"
	)
	// No element exists in empty list
	exists, err := main.StrsetHas(be, l1, "val")
	assert.Nil(t, err, msg)
	assert.False(t, exists, msg)
	exists, err = main.StrsetHas(be, l1, "")
	assert.Nil(t, err, msg)
	assert.False(t, exists, msg)
	// Getting an empty list should return an empty slice
	strset, err = main.StrsetGet(be, l1)
	assert.Nil(t, err, msg)
	assert.Equal(t, []string{}, strset, msg)
	// Inserting to an empty list should work, and after that the element should exist but not another
	err = main.StrsetInsert(be, l1, "val1")
	assert.Nil(t, err, msg)
	exists, err = main.StrsetHas(be, l1, "val1")
	assert.Nil(t, err, msg)
	assert.True(t, exists, msg)
	exists, err = main.StrsetHas(be, l1, "val2")
	assert.Nil(t, err, msg)
	assert.False(t, exists, msg)
	// Inserting to a non-empty list should work, and after that the element should exist but not another
	err = main.StrsetInsert(be, l1, "val2")
	assert.Nil(t, err, msg)
	exists, err = main.StrsetHas(be, l1, "val1")
	assert.Nil(t, err, msg)
	assert.True(t, exists, msg)
	exists, err = main.StrsetHas(be, l1, "val2")
	assert.Nil(t, err, msg)
	assert.True(t, exists, msg)
	exists, err = main.StrsetHas(be, l1, "val3")
	assert.Nil(t, err, msg)
	assert.False(t, exists, msg)
	// Inserting to another list should not affect the first
	err = main.StrsetInsert(be, l2, "valZ")
	assert.Nil(t, err, msg)
	exists, err = main.StrsetHas(be, l1, "valZ")
	assert.Nil(t, err, msg)
	assert.False(t, exists, msg)
	exists, err = main.StrsetHas(be, l2, "valZ")
	assert.Nil(t, err, msg)
	assert.True(t, exists, msg)
	exists, err = main.StrsetHas(be, l2, "val1")
	assert.Nil(t, err, msg)
	assert.False(t, exists, msg)
	exists, err = main.StrsetHas(be, l1, "val1")
	assert.Nil(t, err, msg)
	assert.True(t, exists, msg)
	strset, err = main.StrsetGet(be, l1)
	assert.Nil(t, err, msg)
	assert.Equal(t, []string{"val1", "val2"}, strset, msg)
	strset, err = main.StrsetGet(be, l2)
	assert.Nil(t, err, msg)
	assert.Equal(t, []string{"valZ"}, strset, msg)
	// Inserting the empty string should work
	err = main.StrsetInsert(be, l1, "")
	assert.Nil(t, err, msg)
	exists, err = main.StrsetHas(be, l1, "")
	assert.Nil(t, err, msg)
	assert.True(t, exists, msg)
	// Getting the list should return all elements
	strset, err = main.StrsetGet(be, l1)
	assert.Nil(t, err, msg)
	assert.Equal(t, []string{"", "val1", "val2"}, strset, msg)
	// Removing a non-existing element should work
	err = main.StrsetRemove(be, l1, "val3")
	assert.Nil(t, err, msg)
	exists, err = main.StrsetHas(be, l1, "val1")
	// Removing an existing element should work
	err = main.StrsetRemove(be, l1, "val1")
	assert.Nil(t, err, msg)
	exists, err = main.StrsetHas(be, l1, "val1")
	assert.Nil(t, err, msg)
	assert.False(t, exists, msg)
	exists, err = main.StrsetHas(be, l1, "val2")
	assert.Nil(t, err, msg)
	assert.True(t, exists, msg)
	// Removing the empty string should work
	err = main.StrsetRemove(be, l1, "")
	assert.Nil(t, err, msg)
	exists, err = main.StrsetHas(be, l1, "")
	assert.Nil(t, err, msg)
	assert.False(t, exists, msg)
	// Getting the list should return all elements
	strset, err = main.StrsetGet(be, l1)
	assert.Nil(t, err, msg)
	assert.Equal(t, []string{"val2"}, strset, msg)
	// Removing the last element should work
	err = main.StrsetRemove(be, l1, "val2")
	assert.Nil(t, err, msg)
	exists, err = main.StrsetHas(be, l1, "val2")
	assert.Nil(t, err, msg)
	assert.False(t, exists, msg)
	// Getting the list should return an empty slice
	strset, err = main.StrsetGet(be, l1)
	assert.Nil(t, err, msg)
	assert.Equal(t, []string{}, strset, msg)
}

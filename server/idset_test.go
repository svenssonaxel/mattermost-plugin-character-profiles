package main_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mattermost/mattermost-server/v5/model"

	"axelsvensson.com/mattermost-plugin-character-profiles/server"
)

// This test case is anologous to TestStrset but for the Idset functions
func TestIdset(t *testing.T) {
	// Initialize the backend mock
	be := main.BackendMock{
		IdCounter: new(int),
		KVStore:   map[string][]byte{},
	}
	msg := "TestIdset"
	// Push some values to various lists
	var (
		err    *model.AppError
		strset []string
		l1     = "test1"
		l2     = "test2"
		val0   = "val0aaaaaaaaaaaaaaaaaaaaaa"
		val1   = "val1aaaaaaaaaaaaaaaaaaaaaa"
		val2   = "val2aaaaaaaaaaaaaaaaaaaaaa"
		val3   = "val3aaaaaaaaaaaaaaaaaaaaaa"
		valz   = "valzaaaaaaaaaaaaaaaaaaaaaa"
	)
	// No element exists in empty list
	exists, err := main.IdsetHas(be, l1, val0)
	assert.Nil(t, err, msg)
	assert.False(t, exists, msg)
	exists, err = main.IdsetHas(be, l1, "")
	assert.NotNil(t, err, msg)
	// Getting an empty list should return an empty slice
	strset, err = idsetGet(be, l1)
	assert.Nil(t, err, msg)
	assert.Equal(t, []string{}, strset, msg)
	// Inserting to an empty list should work, and after that the element should exist but not another
	err = main.IdsetInsert(be, l1, val1)
	assert.Nil(t, err, msg)
	exists, err = main.IdsetHas(be, l1, val1)
	assert.Nil(t, err, msg)
	assert.True(t, exists, msg)
	exists, err = main.IdsetHas(be, l1, val2)
	assert.Nil(t, err, msg)
	assert.False(t, exists, msg)
	// Inserting to a non-empty list should work, and after that the element should exist but not another
	err = main.IdsetInsert(be, l1, val2)
	assert.Nil(t, err, msg)
	exists, err = main.IdsetHas(be, l1, val1)
	assert.Nil(t, err, msg)
	assert.True(t, exists, msg)
	exists, err = main.IdsetHas(be, l1, val2)
	assert.Nil(t, err, msg)
	assert.True(t, exists, msg)
	exists, err = main.IdsetHas(be, l1, val3)
	assert.Nil(t, err, msg)
	assert.False(t, exists, msg)
	// Inserting to another list should not affect the first
	err = main.IdsetInsert(be, l2, valz)
	assert.Nil(t, err, msg)
	exists, err = main.IdsetHas(be, l1, valz)
	assert.Nil(t, err, msg)
	assert.False(t, exists, msg)
	exists, err = main.IdsetHas(be, l2, valz)
	assert.Nil(t, err, msg)
	assert.True(t, exists, msg)
	exists, err = main.IdsetHas(be, l2, val1)
	assert.Nil(t, err, msg)
	assert.False(t, exists, msg)
	exists, err = main.IdsetHas(be, l1, val1)
	assert.Nil(t, err, msg)
	assert.True(t, exists, msg)
	strset, err = idsetGet(be, l1)
	assert.Nil(t, err, msg)
	assert.Equal(t, []string{val1, val2}, strset, msg)
	strset, err = idsetGet(be, l2)
	assert.Nil(t, err, msg)
	assert.Equal(t, []string{valz}, strset, msg)
	// Inserting the empty string should not work
	err = main.IdsetInsert(be, l1, "")
	assert.NotNil(t, err, msg)
	// Getting the list should return all elements
	strset, err = idsetGet(be, l1)
	assert.Nil(t, err, msg)
	assert.Equal(t, []string{val1, val2}, strset, msg)
	// Removing a non-existing element should work
	err = main.IdsetRemove(be, l1, val3)
	assert.Nil(t, err, msg)
	exists, err = main.IdsetHas(be, l1, val1)
	// Removing an existing element should work
	err = main.IdsetRemove(be, l1, val1)
	assert.Nil(t, err, msg)
	exists, err = main.IdsetHas(be, l1, val1)
	assert.Nil(t, err, msg)
	assert.False(t, exists, msg)
	exists, err = main.IdsetHas(be, l1, val2)
	assert.Nil(t, err, msg)
	assert.True(t, exists, msg)
	// Removing the empty string should not work
	err = main.IdsetRemove(be, l1, "")
	assert.NotNil(t, err, msg)
	// Checking if the empty string exists should not work
	_, err = main.IdsetHas(be, l1, "")
	assert.NotNil(t, err, msg)
	// Getting the list should return all elements
	strset, err = idsetGet(be, l1)
	assert.Nil(t, err, msg)
	assert.Equal(t, []string{val2}, strset, msg)
	// Removing the last element should work
	err = main.IdsetRemove(be, l1, val2)
	assert.Nil(t, err, msg)
	exists, err = main.IdsetHas(be, l1, val2)
	assert.Nil(t, err, msg)
	assert.False(t, exists, msg)
	// Getting the list should return an empty slice
	strset, err = idsetGet(be, l1)
	assert.Nil(t, err, msg)
	assert.Equal(t, []string{}, strset, msg)
}

// Test for the IdsetIter function.
func TestIdsetIter(t *testing.T) {
	// Initialize the backend mock
	be := main.BackendMock{
		IdCounter: new(int),
		KVStore:   map[string][]byte{},
	}
	msg := "TestIdsetIter"
	// Create an idset with 1000 elements, and also a list of the same elements
	list := []string{}
	for i := 0; i < 1000; i++ {
		id := be.NewId()
		list = append(list, id)
		err := main.IdsetInsert(be, "list", id)
		assert.Nil(t, err, msg)
	}
	// Sort the list
	sort.Strings(list)
	// Check that the elements are the same
	strset, err := idsetGet(be, "list")
	assert.Nil(t, err, msg)
	assert.Equal(t, list, strset, msg)
	// Test use of the beginAfter and maxIterations parameters
	strset, err = idsetGetsubseq(be, "list", list[0], 0)
	assert.Nil(t, err, msg)
	assert.Equal(t, list[1:], strset, msg)
	strset, err = idsetGetsubseq(be, "list", list[0], 1)
	assert.Nil(t, err, msg)
	assert.Equal(t, list[1:2], strset, msg)
	strset, err = idsetGetsubseq(be, "list", list[500], 0)
	assert.Nil(t, err, msg)
	assert.Equal(t, list[501:], strset, msg)
	strset, err = idsetGetsubseq(be, "list", list[500], 1234567)
	assert.Nil(t, err, msg)
	assert.Equal(t, list[501:], strset, msg)
	strset, err = idsetGetsubseq(be, "list", "", 750)
	assert.Nil(t, err, msg)
	assert.Equal(t, list[:750], strset, msg)
	// Test that the iteration stops when the callback returns an error
	strset = []string{}
	err = main.IdsetIter(be, "list", "", 0, func(id string) *model.AppError {
		strset = append(strset, id)
		if id == list[500] {
			return model.NewAppError("TestIdsetIter", "test error", nil, "", 0)
		}
		return nil
	})
	assert.NotNil(t, err, msg)
	assert.Equal(t, list[:501], strset, msg)
}

func idsetGet(be main.Backend, key string) ([]string, *model.AppError) {
	return idsetGetsubseq(be, key, "", 0)
}

func idsetGetsubseq(be main.Backend, key string, beginAfter string, maxIterations int) ([]string, *model.AppError) {
	ret := []string{}
	err := main.IdsetIter(be, key, beginAfter, maxIterations, func(id string) *model.AppError {
		ret = append(ret, id)
		return nil
	})
	return ret, err
}

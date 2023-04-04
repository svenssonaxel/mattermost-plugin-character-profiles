package main

import (
	"encoding/json"
	"sort"

	"github.com/mattermost/mattermost-server/v5/model"
)

// Implementation of named sets of strings stored in the KV store. The
// implementation uses a sorted array of strings to allow efficient
// insertion, removal, and lookup of elements. The array is stored as a JSON
// array in the KV store. Note that this implementation does not scale well
// with large sets.

// Get a sorted string array from the KV store along with the raw JSON value.
func strsetGet(be Backend, key string) ([]string, []byte, *model.AppError) {
	jsonVal, err := be.KVGet(key)
	if err != nil {
		return nil, nil, err
	}
	if jsonVal == nil {
		return []string{}, jsonVal, nil
	}
	contents := []string{}
	jsonErr := json.Unmarshal(jsonVal, &contents)
	if jsonErr != nil {
		return nil, jsonVal, appError("Failed to unmarshal string array.", jsonErr)
	}
	return contents, jsonVal, nil
}

// Get a sorted string array from the KV store.
func StrsetGet(be Backend, key string) ([]string, *model.AppError) {
	contents, _, err := strsetGet(be, key)
	return contents, err
}

// Insert an element into a sorted string array in the KV store unless it is
// already present.
func StrsetInsert(be Backend, key string, element string) *model.AppError {
	oldContents, oldJson, err := strsetGet(be, key)
	if err != nil {
		return err
	}
	i := sort.SearchStrings(oldContents, element)
	if i < len(oldContents) && oldContents[i] == element {
		// Element already present
		return nil
	}
	// Insert element at index i
	newContents := make([]string, len(oldContents)+1)
	copy(newContents, oldContents[:i])
	newContents[i] = element
	copy(newContents[i+1:], oldContents[i:])
	newJson, jsonErr := json.Marshal(newContents)
	if jsonErr != nil {
		return appError("Failed to marshal string array.", jsonErr)
	}
	_, err = be.KVCompareAndSet(key, oldJson, newJson)
	if err != nil {
		return err
	}
	return nil
}

// Remove an element from a sorted string array in the KV store if it is
// present.
func StrsetRemove(be Backend, key string, element string) *model.AppError {
	oldContents, oldJson, err := strsetGet(be, key)
	if err != nil {
		return err
	}
	i := sort.SearchStrings(oldContents, element)
	if i >= len(oldContents) || oldContents[i] != element {
		// Element already absent
		return nil
	}
	// Remove element at index i
	newContents := make([]string, len(oldContents)-1)
	copy(newContents, oldContents[:i])
	copy(newContents[i:], oldContents[i+1:])
	newJson, jsonErr := json.Marshal(newContents)
	if jsonErr != nil {
		return appError("Failed to marshal string array.", jsonErr)
	}
	_, err = be.KVCompareAndSet(key, oldJson, newJson)
	if err != nil {
		return err
	}
	return nil
}

// Check if an element is present in a sorted string array in the KV store.
func StrsetHas(be Backend, key string, element string) (bool, *model.AppError) {
	contents, _, err := strsetGet(be, key)
	if err != nil {
		return false, err
	}
	i := sort.SearchStrings(contents, element)
	return i < len(contents) && contents[i] == element, nil
}

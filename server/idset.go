package main

import (
	"regexp"

	"github.com/mattermost/mattermost-server/v5/model"
)

// Implementation of large, named sets of MM ids stored in the KV store. The MM
// ids are zbase32-encoded UUIDs i.e. 26-character strings of lowercase letters
// and digits. In order to be scalable, the implementation uses Strarr*
// functions to store two types of strring sets:
// - A set of two-character prefixes of MM ids.
// - For each such prefix, a set of MM ids that share that prefix. This set is
//   used for all operations.
// In order to avoid a complex locking scheme, we only enforce the invariant
// that the prefix set is always a superset of, but perhaps does not equal, the
// set of two-character prefixes of the ids in the id sets. This means that
// there may be some prefixes in the prefix set that do not correspond to any
// of the ids in any of the id sets. This is not a problem, because the prefix
// set is only used to iterate over the id sets, and the iteration will simply
// skip over the prefixes that are not present in any of the id sets.

// Iterate over the elements of an id set.
// beginAfter is the last element to skip over, or "" to start at the beginning.
// maxIterations is the maximum number of elements to iterate over, or 0 for no
// limit.
func IdsetIter(be Backend, key string, beginAfter string, maxIterations int, exec func(string) *model.AppError) *model.AppError {
	// Validate the beginAfter element
	beginAfterPrefix := ""
	if beginAfter != "" {
		err := idValidate(beginAfter)
		if err != nil {
			return err
		}
		beginAfterPrefix = beginAfter[:2]
	}
	// Iterate over the prefixes
	prefixes, err := StrsetGet(be, "idp_"+key)
	if err != nil {
		return err
	}
	for _, prefix := range prefixes {
		if prefix < beginAfterPrefix {
			continue
		}
		// Iterate over the ids with this prefix
		ids, err := StrsetGet(be, "id_"+key+"_"+prefix)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if id <= beginAfter {
				continue
			}
			err := exec(id)
			if err != nil {
				return err
			}
			if maxIterations > 0 {
				maxIterations--
				if maxIterations == 0 {
					return nil
				}
			}
		}
	}
	return nil
}

// Insert an element into an id set unless it is already present.
func IdsetInsert(be Backend, key string, element string) *model.AppError {
	// Validate the element
	err := idValidate(element)
	if err != nil {
		return err
	}
	// Insert prefix into the prefix set. We do this first in order to ensure
	// the invariant mentioned above.
	prefix := element[:2]
	err = StrsetInsert(be, "idp_"+key, prefix)
	if err != nil {
		return err
	}
	// Insert element into the id set
	err = StrsetInsert(be, "id_"+key+"_"+prefix, element)
	if err != nil {
		return err
	}
	return nil
}

// Remove an element from an id set if it is present.
func IdsetRemove(be Backend, key string, element string) *model.AppError {
	// Validate the element
	err := idValidate(element)
	if err != nil {
		return err
	}
	// Remove element from the id set if it is present.
	prefix := element[:2]
	err = StrsetRemove(be, "id_"+key+"_"+prefix, element)
	if err != nil {
		return err
	}
	// Without a complex locking scheme we don't know if we can remove the
	// prefix from the prefix set. It presents no problem and violates no
	// invariant to leave it, so we won't bother.
	return nil
}

// Check if an element is present in an id set.
func IdsetHas(be Backend, key string, element string) (bool, *model.AppError) {
	// Validate the element
	err := idValidate(element)
	if err != nil {
		return false, err
	}
	// We don't need to check if the prefix is present in the prefix set. Due to
	// the invariant mentioned above, it always will be if the element is present
	// in the id set.
	prefix := element[:2]
	present, err := StrsetHas(be, "id_"+key+"_"+prefix, element)
	if err != nil {
		return false, err
	}
	return present, nil
}

// Validate an element of an id set.
func idValidate(element string) *model.AppError {
	if regexp.MustCompile(`^[a-z0-9]{26}$`).FindStringSubmatch(element) == nil {
		return appError("Expected a 26-character alphanumeric string", nil)
	}
	return nil
}

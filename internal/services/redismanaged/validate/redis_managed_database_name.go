// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validate

import (
	"fmt"
)

// RedisManagedDatabaseName - validates that the passed interface contains a valid Redis Enterprise Database Name
func RedisManagedDatabaseName(i interface{}, k string) (warnings []string, errors []error) {
	v, ok := i.(string)
	if !ok {
		errors = append(errors, fmt.Errorf("%q expected type of to be string", k))
		return
	}

	if v != "default" {
		errors = append(errors, fmt.Errorf("%q is currently limited to 'default' only, got %v", k, v))
		return
	}

	return
}

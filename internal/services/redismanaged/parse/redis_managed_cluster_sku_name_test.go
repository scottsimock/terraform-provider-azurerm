// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package parse

import (
	"testing"
)

func TestRedisManagedSkuName(t *testing.T) {
	testData := []struct {
		Input    string
		Error    bool
		Expected *RedisManagedCacheSku
	}{
		{
			// empty
			Input: "",
			Error: true,
		},
		{
			// missing sku name and capacity
			Input: "-",
			Error: true,
		},
		{
			// missing sku name and capacity with multi delimiter
			Input: "--",
			Error: true,
		},
		{
			// missing sku name
			Input: "-1",
			Error: true,
		},
		{
			// missing capacity
			Input: "Sku1-",
			Error: true,
		},
		{
			// invalid capacity type
			Input: "Sku1-A",
			Error: true,
		},
		{
			// invalid capacity above int32 type
			Input: "Sku1-2147483648",
			Error: true,
		},
		{
			// valid with ignored extra delimiter
			Input: "skuName-1-",
			Expected: &RedisManagedCacheSku{
				Name:     "skuName",
				Capacity: "1",
			},
		},
		{
			// valid
			Input: "skuName-1",
			Expected: &RedisManagedCacheSku{
				Name:     "skuName",
				Capacity: "1",
			},
		},
		{
			// upper-cased
			Input: "SKUNAME-1",
			Expected: &RedisManagedCacheSku{
				Name:     "SKUNAME",
				Capacity: "1",
			},
		},
	}

	for _, v := range testData {
		t.Logf("[DEBUG] Testing %q", v.Input)

		actual, err := RedisManagedCacheSkuName(v.Input)
		if err != nil {
			if v.Error {
				continue
			}
			t.Fatalf("Expected a value but got an error: %s", err)
		}

		if actual.Name != v.Expected.Name {
			t.Fatalf("Expected %q but got %q for Name", v.Expected.Name, actual.Name)
		}

		if actual.Capacity != v.Expected.Capacity {
			t.Fatalf("Expected %q but got %q for Name", v.Expected.Name, actual.Name)
		}
	}
}

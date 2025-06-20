// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package parse

import (
	"fmt"
	"strconv"
	"strings"
)

// RedisManagedCacheSku type
type RedisManagedCacheSku struct {
	Name     string
	Capacity string
}

// RedisManagedCacheSkuName parses the input string into a RedisManagedCacheSku type
func RedisManagedCacheSkuName(input string) (*RedisManagedCacheSku, error) {
	if len(strings.TrimSpace(input)) == 0 {
		return nil, fmt.Errorf("unable to parse Redis Enterprise Cluster 'sku_name' %q", input)
	}

	skuParts := strings.Split(input, "-")

	if len(skuParts) < 2 {
		return nil, fmt.Errorf("invalid Redis Enterprise Cluster 'sku_name', got %q", input)
	}

	if strings.TrimSpace(skuParts[0]) == "" {
		return nil, fmt.Errorf("invalid Redis Enterprise Cluster 'sku_name' missing 'name' segment, got %q", input)
	}

	if strings.TrimSpace(skuParts[1]) == "" {
		return nil, fmt.Errorf("invalid Redis Enterprise Cluster 'sku_name' missing 'capacity' segment, got %q", input)
	}

	_, err := strconv.ParseInt(skuParts[1], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid Redis Enterprise Cluster 'sku_name', 'capacity' segment must be of type int32 and be a valid int32 value, got %q", skuParts[1])
	}

	redisEnterpriseCacheSku := RedisManagedCacheSku{
		Name:     skuParts[0],
		Capacity: skuParts[1],
	}

	return &redisEnterpriseCacheSku, nil
}

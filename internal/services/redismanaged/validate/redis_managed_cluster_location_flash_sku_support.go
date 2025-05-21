// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validate

import (
	"fmt"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/location"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/azure"
)

// RedisManagedClusterLocationFlashSkuSupport - validates that the passed location supports the flash sku type or not
func RedisManagedClusterLocationFlashSkuSupport(input string) error {
	location := location.Normalize(input)
	invalidLocations := invalidRedisManagedClusterFlashLocations()

	for _, str := range invalidLocations {
		if location == str {
			return fmt.Errorf("%q does not support Redis Enterprise Clusters Flash SKU's. Locations which do not currently support Redis Enterprise Clusters Flash SKU's are [%s]", input, azure.QuotedStringSlice(friendlyInvalidRedisManagedClusterFlashLocations()))
		}
	}

	return nil
}

func invalidRedisManagedClusterFlashLocations() []string {
	locations := friendlyInvalidRedisManagedClusterFlashLocations()
	validFlash := make([]string, 0, len(locations))

	for _, v := range locations {
		validFlash = append(validFlash, location.Normalize(v))
	}

	return validFlash
}

func friendlyInvalidRedisManagedClusterFlashLocations() []string {
	return []string{
		"Australia Southeast",
		"Central US",
		"Central US EUAP",
		"East Asia",
		"UK West",
		"East US 2 EUAP",
		"South India",
	}
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package validate

import (
	"fmt"

	"github.com/hashicorp/go-azure-helpers/resourcemanager/location"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/azure"
)

// RedisManagedClusterLocationZoneSupport - validates that the passed location supports zones or not
func RedisManagedClusterLocationZoneSupport(input string) error {
	location := location.Normalize(input)
	invalidLocations := invalidRedisManagedClusterZoneLocations()

	for _, str := range invalidLocations {
		if location == str {
			return fmt.Errorf("'Zones' are not currently supported in the %s regions, got %q", azure.QuotedStringSlice(friendlyInvalidRedisManagedClusterZoneLocations()), location)
		}
	}

	return nil
}

func invalidRedisManagedClusterZoneLocations() []string {
	locations := friendlyInvalidRedisManagedClusterZoneLocations()
	invalidZone := make([]string, 0, len(locations))

	for _, v := range locations {
		invalidZone = append(invalidZone, location.Normalize(v))
	}

	return invalidZone
}

func friendlyInvalidRedisManagedClusterZoneLocations() []string {
	return []string{
		"Central US EUAP",
		"West US",
		"Australia Southeast",
		"East Asia",
		"UK West",
		"South India",
	}
}

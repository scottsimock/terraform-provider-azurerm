// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package redismanaged_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance"
	"github.com/hashicorp/terraform-provider-azurerm/internal/acceptance/check"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

type RedisManagedClusterResource struct{}

func TestAccRedisManagedCluster_basic(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_redis_enterprise_cluster", "test")
	r := RedisManagedClusterResource{}
	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.basic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
	})
}

func TestAccRedisManagedCluster_requiresImport(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_redis_enterprise_cluster", "test")
	r := RedisManagedClusterResource{}
	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.basic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.RequiresImportErrorStep(r.requiresImport),
	})
}

func TestAccRedisManagedCluster_complete(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_redis_enterprise_cluster", "test")
	r := RedisManagedClusterResource{}
	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.complete(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
				check.That(data.ResourceName).Key("hostname").Exists(),
			),
		},
		data.ImportStep(),
	})
}

func TestAccRedisManagedCluster_update(t *testing.T) {
	data := acceptance.BuildTestData(t, "azurerm_redis_enterprise_cluster", "test")
	r := RedisManagedClusterResource{}

	data.ResourceTest(t, r, []acceptance.TestStep{
		{
			Config: r.basic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
		{
			Config: r.update(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
		{
			Config: r.basic(data),
			Check: acceptance.ComposeTestCheckFunc(
				check.That(data.ResourceName).ExistsInAzure(r),
			),
		},
		data.ImportStep(),
	})
}

func (r RedisManagedClusterResource) Exists(ctx context.Context, client *clients.Client, state *pluginsdk.InstanceState) (*bool, error) {
	id, err := redismanaged.ParseRedisManagedID(state.ID)
	if err != nil {
		return nil, err
	}
	resp, err := client.RedisManaged.Client.Get(ctx, *id)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			return utils.Bool(false), nil
		}
		return nil, fmt.Errorf("retrieving %s: %+v", *id, err)
	}
	return utils.Bool(resp.Model != nil), nil
}

func (r RedisManagedClusterResource) template(data acceptance.TestData) string {
	// I have to hardcode the location because some features are not currently available in all regions
	return fmt.Sprintf(`
provider "azurerm" {
  features {}
}

resource "azurerm_resource_group" "test" {
  name     = "acctestRG-redisEnterprise-%d"
  location = "%s"
}
`, data.RandomInteger, "eastus")
}

func (r RedisManagedClusterResource) basic(data acceptance.TestData) string {
	template := r.template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_redis_enterprise_cluster" "test" {
  name                = "acctest-rec-%d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location

  sku_name = "Enterprise_E1-2"
}
`, template, data.RandomInteger)
}

func (r RedisManagedClusterResource) update(data acceptance.TestData) string {
	template := r.template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_redis_enterprise_cluster" "test" {
  name                = "acctest-rec-%d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location

  sku_name = "Enterprise_E1-2"

  tags = {
    environment = "Production"
  }
}
`, template, data.RandomInteger)
}

func (r RedisManagedClusterResource) requiresImport(data acceptance.TestData) string {
	config := r.basic(data)
	return fmt.Sprintf(`
%s

resource "azurerm_redis_enterprise_cluster" "import" {
  name                = azurerm_redis_enterprise_cluster.test.name
  resource_group_name = azurerm_redis_enterprise_cluster.test.resource_group_name
  location            = azurerm_redis_enterprise_cluster.test.location

  sku_name = azurerm_redis_enterprise_cluster.test.sku_name
}
`, config)
}

func (r RedisManagedClusterResource) complete(data acceptance.TestData) string {
	template := r.template(data)
	return fmt.Sprintf(`
%s

resource "azurerm_redis_enterprise_cluster" "test" {
  name                = "acctest-rec-%d"
  resource_group_name = azurerm_resource_group.test.name
  location            = azurerm_resource_group.test.location

  minimum_tls_version = "1.2"

  sku_name = "EnterpriseFlash_F300-3"
  zones    = ["1", "2", "3"]

  tags = {
    ENV = "Test"
  }
}
`, template, data.RandomInteger)
}

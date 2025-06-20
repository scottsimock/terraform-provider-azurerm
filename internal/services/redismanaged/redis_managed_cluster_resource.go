// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package redismanaged

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/commonschema"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/location"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/tags"
	"github.com/hashicorp/go-azure-helpers/resourcemanager/zones"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/features"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/redismanaged/parse"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/redismanaged/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

func resourceRedisManagedCluster() *pluginsdk.Resource {
	resource := &pluginsdk.Resource{
		Create: resourceRedisManagedClusterCreate,
		Read:   resourceRedisManagedClusterRead,
		Update: resourceRedisManagedClusterUpdate,
		Delete: resourceRedisManagedClusterDelete,
		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := redismanaged.ParseRedisManagedID(id)
			return err
		}),

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(30 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(30 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(30 * time.Minute),
		},

		Schema: map[string]*pluginsdk.Schema{
			"name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.RedisManagedName,
			},

			"resource_group_name": commonschema.ResourceGroupName(),

			"location": commonschema.Location(),

			"sku_name": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.RedisManagedClusterSkuName,
			},

			"zones": commonschema.ZonesMultipleOptionalForceNew(),

			"minimum_tls_version": {
				Type:     pluginsdk.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  string(redismanaged.TlsVersionOnePointTwo),
				ValidateFunc: validation.StringInSlice([]string{
					string(redismanaged.TlsVersionOnePointTwo),
				}, false),
			},

			"hostname": {
				Type:     pluginsdk.TypeString,
				Computed: true,
			},

			"tags": commonschema.Tags(),
		},
	}

	if !features.FivePointOh() {
		resource.Schema["minimum_tls_version"] = &pluginsdk.Schema{
			Type:     pluginsdk.TypeString,
			Optional: true,
			ForceNew: true,
			Default:  string(redismanaged.TlsVersionOnePointTwo),
			ValidateFunc: validation.StringInSlice([]string{
				string(redismanaged.TlsVersionOnePointZero),
				string(redismanaged.TlsVersionOnePointOne),
				string(redismanaged.TlsVersionOnePointTwo),
			}, false),
		}
	}

	return resource
}

func resourceRedisManagedClusterCreate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).RedisManaged.Client
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id := redismanaged.NewRedisManagedID(subscriptionId, d.Get("resource_group_name").(string), d.Get("name").(string))
	if d.IsNewResource() {
		existing, err := client.Get(ctx, id)
		if err != nil {
			if !response.WasNotFound(existing.HttpResponse) {
				return fmt.Errorf("checking for presence of existing %s: %+v", id, err)
			}
		}

		if !response.WasNotFound(existing.HttpResponse) {
			return tf.ImportAsExistsError("azurerm_redis_managed_cluster", id.ID())
		}
	}

	location := location.Normalize(d.Get("location").(string))
	sku := expandRedisManagedClusterSku(d.Get("sku_name").(string))

	// If the sku type is flash check to make sure that the sku is supported in that region
	if strings.Contains(string(sku.Name), "Flash") {
		if err := validate.RedisManagedClusterLocationFlashSkuSupport(location); err != nil {
			return fmt.Errorf("%s: %s", id, err)
		}
	}

	tlsVersion := redismanaged.TlsVersion(d.Get("minimum_tls_version").(string))
	parameters := redismanaged.Cluster{
		Location: location,
		Sku:      sku,
		Properties: &redismanaged.ClusterProperties{
			MinimumTlsVersion: &tlsVersion,
		},
		Tags: tags.Expand(d.Get("tags").(map[string]interface{})),
	}

	if v, ok := d.GetOk("zones"); ok {
		// Zones are currently not supported in these regions
		if err := validate.RedisManagedClusterLocationZoneSupport(location); err != nil {
			return fmt.Errorf("%s: %s", id, err)
		}
		zones := zones.ExpandUntyped(v.(*pluginsdk.Set).List())
		if len(zones) > 0 {
			parameters.Zones = &zones
		}
	}

	if err := client.CreateThenPoll(ctx, id, parameters); err != nil {
		return fmt.Errorf("waiting for creation of %s: %+v", id, err)
	}

	log.Printf("[DEBUG] Waiting for %s to become available..", id)
	stateConf := &pluginsdk.StateChangeConf{
		Pending:    []string{"Creating", "Updating", "Enabling", "Deleting", "Disabling"},
		Target:     []string{"Running"},
		Refresh:    redisManagedClusterStateRefreshFunc(ctx, client, id),
		MinTimeout: 15 * time.Second,
		Timeout:    d.Timeout(pluginsdk.TimeoutCreate),
	}
	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for %s to become available: %+v", id, err)
	}

	d.SetId(id.ID())
	return resourceRedisManagedClusterRead(d, meta)
}

func resourceRedisManagedClusterRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).RedisManaged.Client
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := redismanaged.ParseRedisManagedID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, *id)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			log.Printf("[DEBUG] %s was not found - removing from state!", *id)
			d.SetId("")
			return nil
		}

		return fmt.Errorf("retrieving %s: %+v", id, err)
	}

	d.Set("name", id.RedisManagedName)
	d.Set("resource_group_name", id.ResourceGroupName)

	if model := resp.Model; model != nil {
		d.Set("location", location.Normalize(model.Location))
		if err := d.Set("tags", tags.Flatten(model.Tags)); err != nil {
			return fmt.Errorf("setting `tags`: %+v", err)
		}

		if err := d.Set("sku_name", flattenRedisManagedClusterSku(model.Sku)); err != nil {
			return fmt.Errorf("setting `sku_name`: %+v", err)
		}

		d.Set("zones", zones.FlattenUntyped(model.Zones))
		if props := model.Properties; props != nil {
			d.Set("hostname", props.HostName)

			tlsVersion := ""
			if props.MinimumTlsVersion != nil {
				tlsVersion = string(*props.MinimumTlsVersion)
			}
			d.Set("minimum_tls_version", tlsVersion)
		}
	}

	return nil
}

func resourceRedisManagedClusterUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).RedisManaged.Client
	ctx, cancel := timeouts.ForUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()
	log.Printf("[INFO] preparing arguments for Azure ARM Redis Cache update.")

	id, err := redismanaged.ParseRedisManagedID(d.Id())
	if err != nil {
		return err
	}

	t := d.Get("tags").(map[string]interface{})
	expandedTags := tags.Expand(t)

	parameters := redismanaged.ClusterUpdate{
		Tags: expandedTags,
	}

	if err := client.UpdateThenPoll(ctx, *id, parameters); err != nil {
		return fmt.Errorf("updating %s: %+v", *id, err)
	}

	log.Printf("[DEBUG] Waiting for %s to become available", *id)
	stateConf := &pluginsdk.StateChangeConf{
		Pending:    []string{"Creating", "Updating", "Enabling", "Deleting", "Disabling"},
		Target:     []string{"Running"},
		Refresh:    redisManagedClusterStateRefreshFunc(ctx, client, *id),
		MinTimeout: 15 * time.Second,
		Timeout:    d.Timeout(pluginsdk.TimeoutCreate),
	}

	if _, err = stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for %s to become available: %+v", *id, err)
	}

	return resourceRedisManagedClusterRead(d, meta)
}

func resourceRedisManagedClusterDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).RedisManaged.Client
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := redismanaged.ParseRedisManagedID(d.Id())
	if err != nil {
		return err
	}

	if err := client.DeleteThenPoll(ctx, *id); err != nil {
		return fmt.Errorf("deleting %s: %+v", *id, err)
	}

	return nil
}

func expandRedisManagedClusterSku(v string) redismanaged.Sku {
	redisSku, _ := parse.RedisManagedCacheSkuName(v)
	capacity, _ := strconv.ParseInt(redisSku.Capacity, 10, 32)

	return redismanaged.Sku{
		Name:     redismanaged.SkuName(redisSku.Name),
		Capacity: utils.Int64(capacity),
	}
}

func flattenRedisManagedClusterSku(input redismanaged.Sku) *string {
	var name redismanaged.SkuName
	var capacity int64

	if input.Name != "" {
		name = input.Name
	}

	if input.Capacity != nil {
		capacity = *input.Capacity
	}

	skuName := fmt.Sprintf("%s-%d", name, capacity)

	return &skuName
}

func redisManagedClusterStateRefreshFunc(ctx context.Context, client *redismanaged.RedisManagedClient, id redismanaged.RedisManagedId) pluginsdk.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.Get(ctx, id)
		if err != nil {
			return nil, "", fmt.Errorf("retrieving %s: %+v", id, err)
		}
		if res.Model == nil || res.Model.Properties == nil || res.Model.Properties.ResourceState == nil {
			return nil, "", fmt.Errorf("retrieving %s: model/resourceState was nil", id)
		}

		return res, string(*res.Model.Properties.ResourceState), nil
	}
}

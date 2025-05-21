// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package redismanaged

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-sdk/resource-manager/redisenterprise/2024-10-01/databases"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/redismanaged/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/validation"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
	"github.com/hashicorp/terraform-provider-azurerm/utils"
)

func resourceRedisManagedDatabase() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceRedisManagedDatabaseCreate,
		Read:   resourceRedisManagedDatabaseRead,
		Update: resourceRedisManagedDatabaseUpdate,
		Delete: resourceRedisManagedDatabaseDelete,

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(30 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(30 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(30 * time.Minute),
		},

		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := redismanaged.ParseDatabaseID(id)
			return err
		}),

		// Since update is not currently supported all attribute have to be marked as FORCE NEW
		// until support for Update comes online in the near future
		Schema: redisEnterpriseDatabaseSchema(),
	}
}

func redisEnterpriseDatabaseSchema() map[string]*pluginsdk.Schema {
	s := map[string]*pluginsdk.Schema{
		"name": {
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ForceNew:     true,
			Default:      "default",
			ValidateFunc: validate.RedisManagedDatabaseName,
		},

		"cluster_id": {
			Type:         pluginsdk.TypeString,
			Required:     true,
			ForceNew:     true,
			ValidateFunc: redismanaged.ValidateRedisManagedID,
		},

		"client_protocol": {
			Type:     pluginsdk.TypeString,
			Optional: true,
			ForceNew: true,
			Default:  string(redismanaged.ProtocolEncrypted),
			ValidateFunc: validation.StringInSlice([]string{
				string(redismanaged.ProtocolEncrypted),
				string(redismanaged.ProtocolPlaintext),
			}, false),
		},

		"clustering_policy": {
			Type:     pluginsdk.TypeString,
			Optional: true,
			ForceNew: true,
			Default:  string(redismanaged.ClusteringPolicyOSSCluster),
			ValidateFunc: validation.StringInSlice([]string{
				string(redismanaged.ClusteringPolicyEnterpriseCluster),
				string(redismanaged.ClusteringPolicyOSSCluster),
			}, false),
		},

		"eviction_policy": {
			Type:     pluginsdk.TypeString,
			Optional: true,
			ForceNew: true,
			Default:  string(redismanaged.EvictionPolicyVolatileLRU),
			ValidateFunc: validation.StringInSlice([]string{
				string(redismanaged.EvictionPolicyAllKeysLFU),
				string(redismanaged.EvictionPolicyAllKeysLRU),
				string(redismanaged.EvictionPolicyAllKeysRandom),
				string(redismanaged.EvictionPolicyVolatileLRU),
				string(redismanaged.EvictionPolicyVolatileLFU),
				string(redismanaged.EvictionPolicyVolatileTTL),
				string(redismanaged.EvictionPolicyVolatileRandom),
				string(redismanaged.EvictionPolicyNoEviction),
			}, false),
		},

		"module": {
			Type:     pluginsdk.TypeList,
			Optional: true,
			ForceNew: true,
			MaxItems: 4,
			Elem: &pluginsdk.Resource{
				Schema: map[string]*pluginsdk.Schema{
					"name": {
						Type:     pluginsdk.TypeString,
						Required: true,
						ForceNew: true,
						ValidateFunc: validation.StringInSlice([]string{
							"RedisBloom",
							"RedisTimeSeries",
							"RediSearch",
							"RedisJSON",
						}, false),
					},

					"args": {
						Type:     pluginsdk.TypeString,
						Optional: true,
						ForceNew: true,
						Default:  "",
					},

					"version": {
						Type:     pluginsdk.TypeString,
						Computed: true,
					},
				},
			},
		},

		"linked_database_id": {
			Type:     pluginsdk.TypeSet,
			Optional: true,
			MaxItems: 5,
			Set:      pluginsdk.HashString,
			Elem: &pluginsdk.Schema{
				Type:         pluginsdk.TypeString,
				ValidateFunc: databases.ValidateDatabaseID,
			},
		},

		"linked_database_group_nickname": {
			Type:         pluginsdk.TypeString,
			Optional:     true,
			ForceNew:     true,
			RequiredWith: []string{"linked_database_id"},
		},

		// This attribute is currently in preview and is not returned by the RP
		// "persistence": {
		// 	Type:     pluginsdk.TypeList,
		// 	Optional: true,
		// 	MaxItems: 1,
		// 	Elem: &pluginsdk.Resource{
		// 		Schema: map[string]*pluginsdk.Schema{
		// 			"aof_enabled": {
		// 				Type:     pluginsdk.TypeBool,
		// 				Optional: true,
		// 			},

		// 			"aof_frequency": {
		// 				Type:     pluginsdk.TypeString,
		// 				Optional: true,
		// 				ValidateFunc: validation.StringInSlice([]string{
		// 					string(redismanaged.Ones),
		// 					string(redismanaged.Always),
		// 				}, false),
		// 			},

		// 			"rdb_enabled": {
		// 				Type:     pluginsdk.TypeBool,
		// 				Optional: true,
		// 			},

		// 			"rdb_frequency": {
		// 				Type:     pluginsdk.TypeString,
		// 				Optional: true,
		// 				ValidateFunc: validation.StringInSlice([]string{
		// 					string(redismanaged.Oneh),
		// 					string(redismanaged.Sixh),
		// 					string(redismanaged.OneTwoh),
		// 				}, false),
		// 			},
		// 		},
		// 	},
		// },

		"port": {
			Type:         pluginsdk.TypeInt,
			Optional:     true,
			ForceNew:     true,
			Default:      10000,
			ValidateFunc: validation.IntBetween(0, 65353),
		},

		"primary_access_key": {
			Type:      pluginsdk.TypeString,
			Computed:  true,
			Sensitive: true,
		},

		"secondary_access_key": {
			Type:      pluginsdk.TypeString,
			Computed:  true,
			Sensitive: true,
		},
	}

	return s
}

func resourceRedisManagedDatabaseCreate(d *pluginsdk.ResourceData, meta interface{}) error {
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	client := meta.(*clients.Client).RedisManaged.DatabaseClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	clusterId, err := redismanaged.ParseRedisManagedID(d.Get("cluster_id").(string))
	if err != nil {
		return fmt.Errorf("parsing `cluster_id`: %+v", err)
	}

	id := databases.NewDatabaseID(subscriptionId, clusterId.ResourceGroupName, clusterId.RedisManagedName, d.Get("name").(string))
	if d.IsNewResource() {
		existing, err := client.Get(ctx, id)
		if err != nil {
			if !response.WasNotFound(existing.HttpResponse) {
				return fmt.Errorf("checking for presence of existing %s: %+v", id, err)
			}
		}

		if !response.WasNotFound(existing.HttpResponse) {
			return tf.ImportAsExistsError("azurerm_redis_enterprise_database", id.ID())
		}
	}

	clusteringPolicy := databases.ClusteringPolicy(d.Get("clustering_policy").(string))
	evictionPolicy := databases.EvictionPolicy(d.Get("eviction_policy").(string))
	protocol := databases.Protocol(d.Get("client_protocol").(string))

	oldItems, newItems := d.GetChange("linked_database_id")
	isForceUnlink, data := forceUnlinkItems(oldItems.(*pluginsdk.Set).List(), newItems.(*pluginsdk.Set).List())
	if isForceUnlink {
		if err := forceUnlinkDatabase(d, meta, *data); err != nil {
			return fmt.Errorf("unlinking database error: %+v", err)
		}
	}

	linkedDatabase, err := expandArmGeoLinkedDatabase(d.Get("linked_database_id").(*pluginsdk.Set).List(), id.ID(), d.Get("linked_database_group_nickname").(string))
	if err != nil {
		return fmt.Errorf("Setting geo database for database %s error: %+v", id.ID(), err)
	}

	isGeoEnabled := false
	if linkedDatabase != nil {
		isGeoEnabled = true
	}
	module, err := expandArmDatabaseModuleArray(d.Get("module").([]interface{}), isGeoEnabled)
	if err != nil {
		return fmt.Errorf("setting module error: %+v", err)
	}

	parameters := databases.Database{
		Properties: &databases.DatabaseProperties{
			ClientProtocol:   &protocol,
			ClusteringPolicy: &clusteringPolicy,
			EvictionPolicy:   &evictionPolicy,
			Modules:          module,
			// Persistence:      expandArmDatabasePersistence(d.Get("persistence").([]interface{})),
			GeoReplication: linkedDatabase,
			Port:           utils.Int64(int64(d.Get("port").(int))),
		},
	}

	future, err := client.Create(ctx, id, parameters)
	if err != nil {
		// @tombuildsstuff: investigate moving this above

		// Need to check if this was due to the cluster having the wrong sku
		if strings.Contains(err.Error(), "The value of the parameter 'properties.modules' is invalid") {
			clusterClient := meta.(*clients.Client).RedisManaged.Client
			resp, err := clusterClient.Get(ctx, *clusterId)
			if err != nil {
				return fmt.Errorf("retrieving %s: %+v", *clusterId, err)
			}

			if resp.Model != nil && strings.Contains(strings.ToLower(string(resp.Model.Sku.Name)), "flash") {
				return fmt.Errorf("creating a Redis Enterprise Database with modules in a Redis Enterprise Cluster that has an incompatible Flash SKU type %q - please remove the Redis Enterprise Database modules or change the Redis Enterprise Cluster SKU type %s", string(resp.Model.Sku.Name), id)
			}
		}

		return fmt.Errorf("creating %s: %+v", id, err)
	}

	if err := future.Poller.PollUntilDone(ctx); err != nil {
		return fmt.Errorf("waiting for creation of %s: %+v", id, err)
	}

	d.SetId(id.ID())
	return resourceRedisManagedDatabaseRead(d, meta)
}

func resourceRedisManagedDatabaseRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).RedisManaged.DatabaseClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := databases.ParseDatabaseID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, *id)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			log.Printf("[INFO] Redis Enterprise Database %q does not exist - removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("retrieving %s: %+v", *id, err)
	}

	keysResp, err := client.ListKeys(ctx, *id)
	if err != nil {
		return fmt.Errorf("listing keys for %s: %+v", *id, err)
	}

	d.Set("name", id.DatabaseName)
	clusterId := redismanaged.NewRedisManagedID(id.SubscriptionId, id.ResourceGroupName, id.RedisManagedName)
	d.Set("cluster_id", clusterId.ID())

	if model := resp.Model; model != nil {
		if props := model.Properties; props != nil {
			clientProtocol := ""
			if props.ClientProtocol != nil {
				clientProtocol = string(*props.ClientProtocol)
			}
			d.Set("client_protocol", clientProtocol)

			clusteringPolicy := ""
			if props.ClusteringPolicy != nil {
				clusteringPolicy = string(*props.ClusteringPolicy)
			}
			d.Set("clustering_policy", clusteringPolicy)

			evictionPolicy := ""
			if props.EvictionPolicy != nil {
				evictionPolicy = string(*props.EvictionPolicy)
			}
			d.Set("eviction_policy", evictionPolicy)
			if err := d.Set("module", flattenArmDatabaseModuleArray(props.Modules)); err != nil {
				return fmt.Errorf("setting `module`: %+v", err)
			}
			// if err := d.Set("persistence", flattenArmDatabasePersistence(props.Persistence)); err != nil {
			// 	return fmt.Errorf("setting `persistence`: %+v", err)
			// }
			if geoProps := props.GeoReplication; geoProps != nil {
				if geoProps.GroupNickname != nil {
					d.Set("linked_database_group_nickname", geoProps.GroupNickname)
				}
				if err := d.Set("linked_database_id", flattenArmGeoLinkedDatabase(geoProps.LinkedDatabases)); err != nil {
					return fmt.Errorf("setting `linked_database_id`: %+v", err)
				}
			}
			d.Set("port", props.Port)
		}
	}

	if model := keysResp.Model; model != nil {
		d.Set("primary_access_key", model.PrimaryKey)
		d.Set("secondary_access_key", model.SecondaryKey)
	}

	return nil
}

func resourceRedisManagedDatabaseUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	subscriptionId := meta.(*clients.Client).Account.SubscriptionId
	client := meta.(*clients.Client).RedisManaged.DatabaseClient
	ctx, cancel := timeouts.ForCreateUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	clusterId, err := redismanaged.ParseRedisManagedID(d.Get("cluster_id").(string))
	if err != nil {
		return fmt.Errorf("parsing `cluster_id`: %+v", err)
	}

	id := databases.NewDatabaseID(subscriptionId, clusterId.ResourceGroupName, clusterId.RedisManagedName, d.Get("name").(string))

	clusteringPolicy := databases.ClusteringPolicy(d.Get("clustering_policy").(string))
	evictionPolicy := databases.EvictionPolicy(d.Get("eviction_policy").(string))
	protocol := databases.Protocol(d.Get("client_protocol").(string))

	oldItems, newItems := d.GetChange("linked_database_id")
	isForceUnlink, data := forceUnlinkItems(oldItems.(*pluginsdk.Set).List(), newItems.(*pluginsdk.Set).List())
	if isForceUnlink {
		if err := forceUnlinkDatabase(d, meta, *data); err != nil {
			return fmt.Errorf("unlinking database error: %+v", err)
		}
	}

	linkedDatabase, err := expandArmGeoLinkedDatabase(d.Get("linked_database_id").(*pluginsdk.Set).List(), id.ID(), d.Get("linked_database_group_nickname").(string))
	if err != nil {
		return fmt.Errorf("Setting geo database for database %s error: %+v", id.ID(), err)
	}

	isGeoEnabled := false
	if linkedDatabase != nil {
		isGeoEnabled = true
	}
	module, err := expandArmDatabaseModuleArray(d.Get("module").([]interface{}), isGeoEnabled)
	if err != nil {
		return fmt.Errorf("setting module error: %+v", err)
	}

	parameters := databases.Database{
		Properties: &databases.DatabaseProperties{
			ClientProtocol:   &protocol,
			ClusteringPolicy: &clusteringPolicy,
			EvictionPolicy:   &evictionPolicy,
			Modules:          module,
			// Persistence:      expandArmDatabasePersistence(d.Get("persistence").([]interface{})),
			GeoReplication: linkedDatabase,
			Port:           utils.Int64(int64(d.Get("port").(int))),
		},
	}

	if err := client.CreateThenPoll(ctx, id, parameters); err != nil {
		return fmt.Errorf("updatig %s: %+v", id, err)
	}

	d.SetId(id.ID())
	return resourceRedisManagedDatabaseRead(d, meta)
}

func resourceRedisManagedDatabaseDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).RedisManaged.DatabaseClient
	clusterClient := meta.(*clients.Client).RedisManaged.Client
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := databases.ParseDatabaseID(d.Id())
	if err != nil {
		return err
	}

	dbId := databases.NewDatabaseID(id.SubscriptionId, id.ResourceGroupName, id.RedisManagedName, id.DatabaseName)
	clusterId := redismanaged.NewRedisManagedID(id.SubscriptionId, id.ResourceGroupName, id.RedisManagedName)

	if _, err := client.Delete(ctx, *id); err != nil {
		return fmt.Errorf("deleting %s: %+v", id, err)
	}

	// can't use the poll since cluster deletion also deletes the default database, which will case db deletion failure
	stateConf := &pluginsdk.StateChangeConf{
		Pending:                   []string{"found"},
		Target:                    []string{"clusterNotFound", "dbNotFound"},
		Refresh:                   redisEnterpriseDatabaseDeleteRefreshFunc(ctx, client, clusterClient, clusterId, dbId),
		ContinuousTargetOccurence: 3,
		Timeout:                   d.Timeout(pluginsdk.TimeoutDelete),
	}

	if _, err := stateConf.WaitForStateContext(ctx); err != nil {
		return fmt.Errorf("waiting for deletion %s: %+v", id, err)
	}

	return nil
}

func redisEnterpriseDatabaseDeleteRefreshFunc(ctx context.Context, databaseClient *databases.DatabasesClient, clusterClient *redismanaged.RedisManagedClient, clusterId redismanaged.RedisManagedId, databaseId databases.DatabaseId) pluginsdk.StateRefreshFunc {
	return func() (interface{}, string, error) {
		cluster, err := clusterClient.Get(ctx, clusterId)
		if err != nil {
			if response.WasNotFound(cluster.HttpResponse) {
				return "clusterNotFound", "clusterNotFound", nil
			}
		}
		db, err := databaseClient.Get(ctx, databaseId)
		if err != nil {
			if response.WasNotFound(db.HttpResponse) {
				return "dbNotFound", "dbNotFound", nil
			}
		}
		return db, "found", nil
	}
}

func expandArmDatabaseModuleArray(input []interface{}, isGeoEnabled bool) (*[]databases.Module, error) {
	results := make([]databases.Module, 0)

	for _, item := range input {
		v := item.(map[string]interface{})
		moduleName := v["name"].(string)
		if moduleName != "RediSearch" && moduleName != "RedisJSON" && isGeoEnabled {
			return nil, fmt.Errorf("Only RediSearch and RedisJSON modules are allowed with geo-replication")
		}
		results = append(results, databases.Module{
			Name: moduleName,
			Args: utils.String(v["args"].(string)),
		})
	}
	return &results, nil
}

// Persistence is currently preview and does not return from the RP but will be fully supported in the near future
// func expandArmDatabasePersistence(input []interface{}) *redismanaged.Persistence {
// 	if len(input) == 0 {
// 		return nil
// 	}
// 	v := input[0].(map[string]interface{})
// 	return &redismanaged.Persistence{
// 		AofEnabled:   utils.Bool(v["aof_enabled"].(bool)),
// 		AofFrequency: redismanaged.AofFrequency(v["aof_frequency"].(string)),
// 		RdbEnabled:   utils.Bool(v["rdb_enabled"].(bool)),
// 		RdbFrequency: redismanaged.RdbFrequency(v["rdb_frequency"].(string)),
// 	}
// }

func flattenArmDatabaseModuleArray(input *[]databases.Module) []interface{} {
	results := make([]interface{}, 0)
	if input == nil {
		return results
	}

	for _, item := range *input {
		args := ""
		if item.Args != nil {
			args = *item.Args
			// new behavior if you do not pass args the RP sets the args to "PARTITIONS AUTO" by default
			// (for RediSearch) which causes the database to be force new on every plan after creation
			// feels like an RP bug, but I added this workaround...
			// NOTE: You also cannot set the args to PARTITIONS AUTO by default else you will get an error on create:
			// Code="InvalidRequestBody" Message="The value of the parameter 'properties.modules' is invalid."
			if strings.EqualFold(args, "PARTITIONS AUTO") {
				args = ""
			}
		}

		var version string
		if item.Version != nil {
			version = *item.Version
		}

		results = append(results, map[string]interface{}{
			"name":    item.Name,
			"args":    args,
			"version": version,
		})
	}

	return results
}

func expandArmGeoLinkedDatabase(inputId []interface{}, parentDBId string, inputGeoName string) (*databases.DatabasePropertiesGeoReplication, error) {
	idList := make([]databases.LinkedDatabase, 0)
	if len(inputId) == 0 {
		return nil, nil
	}
	isParentDbIncluded := false

	for _, id := range inputId {
		if id.(string) == parentDBId {
			isParentDbIncluded = true
		}
		idList = append(idList, databases.LinkedDatabase{
			Id: utils.String(id.(string)),
		})
	}
	if isParentDbIncluded {
		return &databases.DatabasePropertiesGeoReplication{
			LinkedDatabases: &idList,
			GroupNickname:   utils.String(inputGeoName),
		}, nil
	}

	return nil, fmt.Errorf("linked database list must include database ID: %s", parentDBId)
}

func flattenArmGeoLinkedDatabase(inputDB *[]databases.LinkedDatabase) []string {
	results := make([]string, 0)

	if inputDB == nil {
		return results
	}

	for _, item := range *inputDB {
		if item.Id != nil {
			results = append(results, *item.Id)
		}
	}
	return results
}

func forceUnlinkItems(oldItemList []interface{}, newItemList []interface{}) (bool, *[]string) {
	newItems := make(map[string]bool)
	forceUnlinkList := make([]string, 0)
	for _, newItem := range newItemList {
		newItems[newItem.(string)] = true
	}

	for _, oldItem := range oldItemList {
		if !newItems[oldItem.(string)] {
			forceUnlinkList = append(forceUnlinkList, oldItem.(string))
		}
	}
	if len(forceUnlinkList) > 0 {
		return true, &forceUnlinkList
	}
	return false, nil
}

func forceUnlinkDatabase(d *pluginsdk.ResourceData, meta interface{}, unlinkedDbRaw []string) error {
	client := meta.(*clients.Client).RedisManaged.DatabaseClient
	ctx, cancel := timeouts.ForUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()
	log.Printf("[INFO]Preparing to unlink a linked database")

	id, err := databases.ParseDatabaseID(d.Id())
	if err != nil {
		return err
	}

	parameters := databases.ForceUnlinkParameters{
		Ids: unlinkedDbRaw,
	}

	if err := client.ForceUnlinkThenPoll(ctx, *id, parameters); err != nil {
		return fmt.Errorf("force unlinking from database %s error: %+v", id, err)
	}

	return nil
}

// Persistence is currently preview and does not return from the RP but will be fully supported in the near future
// func flattenArmDatabasePersistence(input *redismanaged.Persistence) []interface{} {
// 	if input == nil {
// 		return make([]interface{}, 0)
// 	}

// 	var aofEnabled bool
// 	if input.AofEnabled != nil {
// 		aofEnabled = *input.AofEnabled
// 	}

// 	var aofFrequency redismanaged.AofFrequency
// 	if input.AofFrequency != "" {
// 		aofFrequency = input.AofFrequency
// 	}

// 	var rdbEnabled bool
// 	if input.RdbEnabled != nil {
// 		rdbEnabled = *input.RdbEnabled
// 	}

// 	var rdbFrequency redismanaged.RdbFrequency
// 	if input.RdbFrequency != "" {
// 		rdbFrequency = input.RdbFrequency
// 	}

// 	return []interface{}{
// 		map[string]interface{}{
// 			"aof_enabled":   aofEnabled,
// 			"aof_frequency": aofFrequency,
// 			"rdb_enabled":   rdbEnabled,
// 			"rdb_frequency": rdbFrequency,
// 		},
// 	}
// }

/*
Provides a resource to create a clickhouse account_permission

Example Usage

```hcl
resource "tencentcloud_clickhouse_account_permission" "account_permission_all_database" {
  instance_id = "cdwch-xxxxxx"
  cluster = "default_cluster"
  user_name = "user1"
  all_database = true
  global_privileges = ["SELECT", "ALTER"]
}

resource "tencentcloud_clickhouse_account_permission" "account_permission_not_all_database" {
	instance_id = "cdwch-xxxxxx"
  cluster = "default_cluster"
  user_name = "user2"
  all_database = false
  database_privilege_list {
    database_name = "xxxxxx"
    database_privileges = ["SELECT", "ALTER"]
  }
}
```

Import

clickhouse account_permission can be imported using the id, e.g.

```
terraform import tencentcloud_clickhouse_account_permission.account_permission ${instanceId}#${cluster}#${userName}
```
*/
package tencentcloud

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	clickhouse "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdwch/v20200915"
	"github.com/tencentcloudstack/terraform-provider-tencentcloud/tencentcloud/internal/helper"
)

func resourceTencentCloudClickhouseAccountPermission() *schema.Resource {
	return &schema.Resource{
		Create: resourceTencentCloudClickhouseAccountPermissionCreate,
		Read:   resourceTencentCloudClickhouseAccountPermissionRead,
		Update: resourceTencentCloudClickhouseAccountPermissionUpdate,
		Delete: resourceTencentCloudClickhouseAccountPermissionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"instance_id": {
				Required:    true,
				Type:        schema.TypeString,
				Description: "Instance id.",
			},

			"cluster": {
				Required:    true,
				Type:        schema.TypeString,
				Description: "Cluster name.",
			},

			"user_name": {
				Required:    true,
				Type:        schema.TypeString,
				Description: "User name.",
			},

			"all_database": {
				Required:    true,
				Type:        schema.TypeBool,
				Description: "Whether all database tables.",
			},

			"global_privileges": {
				Optional: true,
				Type:     schema.TypeSet,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "Global privileges.",
			},

			"database_privilege_list": {
				Optional:    true,
				Type:        schema.TypeList,
				Description: "Database privilege list.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"database_name": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Database name.",
						},
						"database_privileges": {
							Type: schema.TypeSet,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
							Optional:    true,
							Description: "Database privileges. Valid valuse: SELECT, INSERT_ALL, ALTER, TRUNCATE, DROP_TABLE, CREATE_TABLE, DROP_DATABASE.",
						},
						"table_privilege_list": {
							Type:        schema.TypeList,
							Optional:    true,
							Description: "Table privilege list.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"table_name": {
										Type:        schema.TypeString,
										Required:    true,
										Description: "Table name.",
									},
									"table_privileges": {
										Type: schema.TypeSet,
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
										Required:    true,
										Description: "Table privileges. Valid values: SELECT, INSERT_ALL, ALTER, TRUNCATE, DROP_TABLE.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceTencentCloudClickhouseAccountPermissionCreate(d *schema.ResourceData, meta interface{}) error {
	defer logElapsed("resource.tencentcloud_clickhouse_account_permission.create")()
	defer inconsistentCheck(d, meta)()

	logId := getLogId(contextNil)

	var (
		request    = clickhouse.NewModifyUserNewPrivilegeRequest()
		instanceId = d.Get("instance_id").(string)
		userName   = d.Get("user_name").(string)
		cluster    = d.Get("cluster").(string)
	)

	request.InstanceId = helper.String(instanceId)
	request.Cluster = helper.String(cluster)
	request.UserName = helper.String(userName)
	request.AllDatabase = helper.Bool(d.Get("all_database").(bool))

	if v, ok := d.GetOk("global_privileges"); ok {
		globalPrivilegesSet := v.(*schema.Set).List()
		for i := range globalPrivilegesSet {
			globalPrivileges := globalPrivilegesSet[i].(string)
			request.GlobalPrivileges = append(request.GlobalPrivileges, &globalPrivileges)
		}
	}

	if v, ok := d.GetOk("database_privilege_list"); ok {
		for _, item := range v.([]interface{}) {
			dMap := item.(map[string]interface{})
			databasePrivilegeInfo := clickhouse.DatabasePrivilegeInfo{}
			if v, ok := dMap["database_name"]; ok {
				databasePrivilegeInfo.DatabaseName = helper.String(v.(string))
			}
			if v, ok := dMap["database_privileges"]; ok {
				databasePrivilegesSet := v.(*schema.Set).List()
				for i := range databasePrivilegesSet {
					databasePrivileges := databasePrivilegesSet[i].(string)
					databasePrivilegeInfo.DatabasePrivileges = append(databasePrivilegeInfo.DatabasePrivileges, &databasePrivileges)
				}
			}
			if v, ok := dMap["table_privilege_list"]; ok {
				for _, item := range v.([]interface{}) {
					tablePrivilegeListMap := item.(map[string]interface{})
					tablePrivilegeInfo := clickhouse.TablePrivilegeInfo{}
					if v, ok := tablePrivilegeListMap["table_name"]; ok {
						tablePrivilegeInfo.TableName = helper.String(v.(string))
					}
					if v, ok := tablePrivilegeListMap["table_privileges"]; ok {
						tablePrivilegesSet := v.(*schema.Set).List()
						for i := range tablePrivilegesSet {
							tablePrivileges := tablePrivilegesSet[i].(string)
							tablePrivilegeInfo.TablePrivileges = append(tablePrivilegeInfo.TablePrivileges, &tablePrivileges)
						}
					}
					databasePrivilegeInfo.TablePrivilegeList = append(databasePrivilegeInfo.TablePrivilegeList, &tablePrivilegeInfo)
				}
			}
			request.DatabasePrivilegeList = append(request.DatabasePrivilegeList, &databasePrivilegeInfo)
		}
	}

	err := resource.Retry(writeRetryTimeout, func() *resource.RetryError {
		result, e := meta.(*TencentCloudClient).apiV3Conn.UseCdwchClient().ModifyUserNewPrivilege(request)
		if e != nil {
			return retryError(e)
		} else {
			log.Printf("[DEBUG]%s api[%s] success, request body [%s], response body [%s]\n", logId, request.GetAction(), request.ToJsonString(), result.ToJsonString())
		}
		return nil
	})
	if err != nil {
		log.Printf("[CRITAL]%s create cdwch accountPermission failed, reason:%+v", logId, err)
		return err
	}

	d.SetId(instanceId + FILED_SP + cluster + FILED_SP + userName)

	return resourceTencentCloudClickhouseAccountPermissionRead(d, meta)
}

func resourceTencentCloudClickhouseAccountPermissionRead(d *schema.ResourceData, meta interface{}) error {
	defer logElapsed("resource.tencentcloud_clickhouse_account_permission.read")()
	defer inconsistentCheck(d, meta)()

	logId := getLogId(contextNil)
	ctx := context.WithValue(context.TODO(), logIdKey, logId)
	service := CdwchService{client: meta.(*TencentCloudClient).apiV3Conn}

	idSplit := strings.Split(d.Id(), FILED_SP)
	if len(idSplit) != 3 {
		return fmt.Errorf("tencentcloud_clickhouse_account id is broken, id is %s", d.Id())
	}

	accountPermission, err := service.DescribeCdwchAccountPermission(ctx, idSplit[0], idSplit[1], idSplit[2])
	if err != nil {
		return err
	}

	if accountPermission == nil {
		d.SetId("")
		log.Printf("[WARN]%s resource `CdwchAccountPermission` [%s] not found, please check if it has been deleted.\n", logId, d.Id())
		return nil
	}

	if accountPermission.InstanceId != nil {
		_ = d.Set("instance_id", accountPermission.InstanceId)
	}

	if accountPermission.Cluster != nil {
		_ = d.Set("cluster", accountPermission.Cluster)
	}

	if accountPermission.UserName != nil {
		_ = d.Set("user_name", accountPermission.UserName)
	}

	if accountPermission.AllDatabase != nil {
		_ = d.Set("all_database", accountPermission.AllDatabase)
	}

	if accountPermission.GlobalPrivileges != nil {
		_ = d.Set("global_privileges", accountPermission.GlobalPrivileges)
	}

	if accountPermission.DatabasePrivilegeList != nil {
		databasePrivilegeListList := []interface{}{}
		for _, databasePrivilegeList := range accountPermission.DatabasePrivilegeList {
			databasePrivilegeListMap := map[string]interface{}{}

			if databasePrivilegeList.DatabaseName != nil {
				databasePrivilegeListMap["database_name"] = databasePrivilegeList.DatabaseName
			}

			if databasePrivilegeList.DatabasePrivileges != nil {
				databasePrivilegeListMap["database_privileges"] = databasePrivilegeList.DatabasePrivileges
			}

			if databasePrivilegeList.TablePrivilegeList != nil {
				tablePrivilegeListList := []interface{}{}
				for _, tablePrivilegeList := range databasePrivilegeList.TablePrivilegeList {
					tablePrivilegeListMap := map[string]interface{}{}

					if tablePrivilegeList.TableName != nil {
						tablePrivilegeListMap["table_name"] = tablePrivilegeList.TableName
					}

					if tablePrivilegeList.TablePrivileges != nil {
						tablePrivilegeListMap["table_privileges"] = tablePrivilegeList.TablePrivileges
					}

					tablePrivilegeListList = append(tablePrivilegeListList, tablePrivilegeListMap)
				}

				databasePrivilegeListMap["table_privilege_list"] = []interface{}{tablePrivilegeListList}
			}

			databasePrivilegeListList = append(databasePrivilegeListList, databasePrivilegeListMap)
		}

		_ = d.Set("database_privilege_list", databasePrivilegeListList)

	}

	return nil
}

func resourceTencentCloudClickhouseAccountPermissionUpdate(d *schema.ResourceData, meta interface{}) error {
	defer logElapsed("resource.tencentcloud_clickhouse_account_permission.update")()
	defer inconsistentCheck(d, meta)()

	logId := getLogId(contextNil)

	request := clickhouse.NewModifyUserNewPrivilegeRequest()

	idSplit := strings.Split(d.Id(), FILED_SP)
	if len(idSplit) != 3 {
		return fmt.Errorf("tencentcloud_clickhouse_account id is broken, id is %s", d.Id())
	}
	request.InstanceId = helper.String(idSplit[0])
	request.Cluster = helper.String(idSplit[1])
	request.UserName = helper.String(idSplit[2])

	immutableArgs := []string{"instance_id", "cluster", "user_name"}

	for _, v := range immutableArgs {
		if d.HasChange(v) {
			return fmt.Errorf("argument `%s` cannot be changed", v)
		}
	}

	request.AllDatabase = helper.Bool(d.Get("all_database").(bool))

	if d.HasChange("global_privileges") {
		if v, ok := d.GetOk("global_privileges"); ok {
			globalPrivilegesSet := v.(*schema.Set).List()
			for i := range globalPrivilegesSet {
				globalPrivileges := globalPrivilegesSet[i].(string)
				request.GlobalPrivileges = append(request.GlobalPrivileges, &globalPrivileges)
			}
		}
	}

	if d.HasChange("database_privilege_list") {
		if v, ok := d.GetOk("database_privilege_list"); ok {
			for _, item := range v.([]interface{}) {
				dMap := item.(map[string]interface{})
				databasePrivilegeInfo := clickhouse.DatabasePrivilegeInfo{}
				if v, ok := dMap["database_name"]; ok {
					databasePrivilegeInfo.DatabaseName = helper.String(v.(string))
				}
				if v, ok := dMap["database_privileges"]; ok {
					databasePrivilegesSet := v.(*schema.Set).List()
					for i := range databasePrivilegesSet {
						databasePrivileges := databasePrivilegesSet[i].(string)
						databasePrivilegeInfo.DatabasePrivileges = append(databasePrivilegeInfo.DatabasePrivileges, &databasePrivileges)
					}
				}
				if v, ok := dMap["table_privilege_list"]; ok {
					for _, item := range v.([]interface{}) {
						tablePrivilegeListMap := item.(map[string]interface{})
						tablePrivilegeInfo := clickhouse.TablePrivilegeInfo{}
						if v, ok := tablePrivilegeListMap["table_name"]; ok {
							tablePrivilegeInfo.TableName = helper.String(v.(string))
						}
						if v, ok := tablePrivilegeListMap["table_privileges"]; ok {
							tablePrivilegesSet := v.(*schema.Set).List()
							for i := range tablePrivilegesSet {
								tablePrivileges := tablePrivilegesSet[i].(string)
								tablePrivilegeInfo.TablePrivileges = append(tablePrivilegeInfo.TablePrivileges, &tablePrivileges)
							}
						}
						databasePrivilegeInfo.TablePrivilegeList = append(databasePrivilegeInfo.TablePrivilegeList, &tablePrivilegeInfo)
					}
				}
				request.DatabasePrivilegeList = append(request.DatabasePrivilegeList, &databasePrivilegeInfo)
			}
		}
	}

	err := resource.Retry(writeRetryTimeout, func() *resource.RetryError {
		result, e := meta.(*TencentCloudClient).apiV3Conn.UseCdwchClient().ModifyUserNewPrivilege(request)
		if e != nil {
			return retryError(e)
		} else {
			log.Printf("[DEBUG]%s api[%s] success, request body [%s], response body [%s]\n", logId, request.GetAction(), request.ToJsonString(), result.ToJsonString())
		}
		return nil
	})
	if err != nil {
		log.Printf("[CRITAL]%s update cdwch accountPermission failed, reason:%+v", logId, err)
		return err
	}

	return resourceTencentCloudClickhouseAccountPermissionRead(d, meta)
}

func resourceTencentCloudClickhouseAccountPermissionDelete(d *schema.ResourceData, meta interface{}) error {
	defer logElapsed("resource.tencentcloud_clickhouse_account_permission.delete")()
	defer inconsistentCheck(d, meta)()

	logId := getLogId(contextNil)
	ctx := context.WithValue(context.TODO(), logIdKey, logId)

	idSplit := strings.Split(d.Id(), FILED_SP)
	if len(idSplit) != 3 {
		return fmt.Errorf("tencentcloud_clickhouse_account id is broken, id is %s", d.Id())
	}

	service := CdwchService{client: meta.(*TencentCloudClient).apiV3Conn}
	err := service.DescribeCkSqlApis(ctx, idSplit[0], idSplit[1], idSplit[2], DESCRIBE_CK_SQL_APIS_REVOKE_CLUSTER_USER)
	if err != nil {
		return err
	}

	return nil
}

/*
Provides a resource to create a sqlserver config_database_ct

Example Usage

```hcl
data "tencentcloud_availability_zones_by_product" "zones" {
  product = "sqlserver"
}

resource "tencentcloud_vpc" "vpc" {
  name       = "vpc-example"
  cidr_block = "10.0.0.0/16"
}

resource "tencentcloud_subnet" "subnet" {
  availability_zone = data.tencentcloud_availability_zones_by_product.zones.zones.4.name
  name              = "subnet-example"
  vpc_id            = tencentcloud_vpc.vpc.id
  cidr_block        = "10.0.0.0/16"
  is_multicast      = false
}

resource "tencentcloud_security_group" "security_group" {
  name        = "sg-example"
  description = "desc."
}

resource "tencentcloud_sqlserver_basic_instance" "example" {
  name                   = "tf-example"
  availability_zone      = data.tencentcloud_availability_zones_by_product.zones.zones.4.name
  charge_type            = "POSTPAID_BY_HOUR"
  vpc_id                 = tencentcloud_vpc.vpc.id
  subnet_id              = tencentcloud_subnet.subnet.id
  project_id             = 0
  memory                 = 4
  storage                = 100
  cpu                    = 2
  machine_type           = "CLOUD_PREMIUM"
  maintenance_week_set   = [1, 2, 3]
  maintenance_start_time = "09:00"
  maintenance_time_span  = 3
  security_groups        = [tencentcloud_security_group.security_group.id]

  tags = {
    "test" = "test"
  }
}

resource "tencentcloud_sqlserver_db" "example" {
  instance_id = tencentcloud_sqlserver_basic_instance.example.id
  name        = "tf_example_db"
  charset     = "Chinese_PRC_BIN"
  remark      = "test-remark"
}

resource "tencentcloud_sqlserver_config_database_ct" "example" {
  instance_id          = tencentcloud_sqlserver_basic_instance.example.id
  db_name              = tencentcloud_sqlserver_db.example.name
  modify_type          = "disable"
  change_retention_day = 7
}
```

Import

sqlserver tencentcloud_sqlserver_config_database_ct can be imported using the id, e.g.

```
terraform import tencentcloud_sqlserver_config_database_ct.example mssql-i9ma6oy7#tf_example_db
```
*/
package tencentcloud

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	sqlserver "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/sqlserver/v20180328"
	"github.com/tencentcloudstack/terraform-provider-tencentcloud/tencentcloud/internal/helper"
)

func resourceTencentCloudSqlserverConfigDatabaseCT() *schema.Resource {
	return &schema.Resource{
		Create: resourceTencentCloudSqlserverConfigDatabaseCTCreate,
		Read:   resourceTencentCloudSqlserverConfigDatabaseCTRead,
		Update: resourceTencentCloudSqlserverConfigDatabaseCTUpdate,
		Delete: resourceTencentCloudSqlserverConfigDatabaseCTDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"db_name": {
				Required:    true,
				Type:        schema.TypeString,
				Description: "database name.",
			},
			"modify_type": {
				Required:    true,
				Type:        schema.TypeString,
				Description: "Enable or disable CT. Valid values: enable, disable.",
			},
			"instance_id": {
				Required:    true,
				Type:        schema.TypeString,
				Description: "Instance ID.",
			},
			"change_retention_day": {
				Optional:    true,
				Computed:    true,
				Type:        schema.TypeInt,
				Description: "Retention period (in days) of change tracking information when CT is enabled. Value range: 3-30. Default value: 3.",
			},
		},
	}
}

func resourceTencentCloudSqlserverConfigDatabaseCTCreate(d *schema.ResourceData, meta interface{}) error {
	defer logElapsed("resource.tencentcloud_sqlserver_config_database_c_t.create")()
	defer inconsistentCheck(d, meta)()

	var (
		instanceId string
		dbName     string
	)

	if v, ok := d.GetOk("instance_id"); ok {
		instanceId = v.(string)
	}

	if v, ok := d.GetOk("db_name"); ok {
		dbName = v.(string)
	}

	d.SetId(strings.Join([]string{instanceId, dbName}, FILED_SP))

	return resourceTencentCloudSqlserverConfigDatabaseCTUpdate(d, meta)
}

func resourceTencentCloudSqlserverConfigDatabaseCTRead(d *schema.ResourceData, meta interface{}) error {
	defer logElapsed("resource.tencentcloud_sqlserver_config_database_c_t.read")()
	defer inconsistentCheck(d, meta)()

	var (
		logId              = getLogId(contextNil)
		ctx                = context.WithValue(context.TODO(), logIdKey, logId)
		service            = SqlserverService{client: meta.(*TencentCloudClient).apiV3Conn}
		Name               string
		modifyType         string
		retentionPeriod    string
		changeRetentionDay int
	)

	idSplit := strings.Split(d.Id(), FILED_SP)
	if len(idSplit) != 2 {
		return fmt.Errorf("id is broken,%s", idSplit)
	}
	instanceId := idSplit[0]
	dbName := idSplit[1]

	configDatabaseCT, err := service.DescribeSqlserverConfigDatabaseCTById(ctx, instanceId)
	if err != nil {
		return err
	}

	if configDatabaseCT == nil {
		d.SetId("")
		log.Printf("[WARN]%s resource `SqlserverConfigDatabaseCT` [%s] not found, please check if it has been deleted.\n", logId, d.Id())
		return nil
	}

	for _, i := range configDatabaseCT {
		if *i.Name == dbName {
			if *i.IsDbChainingOn == "0" {
				modifyType = "disable"
			} else {
				modifyType = "enable"
			}
			Name = *i.Name
			retentionPeriod = *i.RetentionPeriod
			break
		}
	}

	_ = d.Set("instance_id", instanceId)
	_ = d.Set("db_name", Name)
	_ = d.Set("modify_type", modifyType)
	changeRetentionDay, _ = strconv.Atoi(retentionPeriod)
	_ = d.Set("change_retention_day", changeRetentionDay)

	return nil
}

func resourceTencentCloudSqlserverConfigDatabaseCTUpdate(d *schema.ResourceData, meta interface{}) error {
	defer logElapsed("resource.tencentcloud_sqlserver_config_database_c_t.update")()
	defer inconsistentCheck(d, meta)()

	var (
		logId       = getLogId(contextNil)
		request     = sqlserver.NewModifyDatabaseCTRequest()
		flowRequest = sqlserver.NewDescribeFlowStatusRequest()
		flowId      int64
	)

	idSplit := strings.Split(d.Id(), FILED_SP)
	if len(idSplit) != 2 {
		return fmt.Errorf("id is broken,%s", idSplit)
	}
	instanceId := idSplit[0]
	dbName := idSplit[1]

	if v, ok := d.GetOk("modify_type"); ok {
		request.ModifyType = helper.String(v.(string))
	}

	if v, ok := d.GetOk("change_retention_day"); ok {
		request.ChangeRetentionDay = helper.IntInt64(v.(int))
	}

	request.InstanceId = &instanceId
	dbNames := make([]*string, 0)
	dbNames = append(dbNames, &dbName)
	request.DBNames = dbNames

	err := resource.Retry(writeRetryTimeout, func() *resource.RetryError {
		result, e := meta.(*TencentCloudClient).apiV3Conn.UseSqlserverClient().ModifyDatabaseCT(request)
		if e != nil {
			return retryError(e)
		} else {
			log.Printf("[DEBUG]%s api[%s] success, request body [%s], response body [%s]\n", logId, request.GetAction(), request.ToJsonString(), result.ToJsonString())
		}

		if result == nil {
			e = fmt.Errorf("sqlserver configDatabaseCT not exists")
			return resource.NonRetryableError(e)
		}

		flowId = *result.Response.FlowId
		return nil
	})

	if err != nil {
		log.Printf("[CRITAL]%s update sqlserver configDatabaseCT failed, reason:%+v", logId, err)
		return err
	}

	flowRequest.FlowId = &flowId
	err = resource.Retry(10*writeRetryTimeout, func() *resource.RetryError {
		result, e := meta.(*TencentCloudClient).apiV3Conn.UseSqlserverClient().DescribeFlowStatus(flowRequest)
		if e != nil {
			return retryError(e)
		}
		if *result.Response.Status == SQLSERVER_TASK_SUCCESS {
			return nil
		} else if *result.Response.Status == SQLSERVER_TASK_RUNNING {
			return resource.RetryableError(fmt.Errorf("sqlserver configDatabaseCT status is running"))
		} else if *result.Response.Status == int64(SQLSERVER_TASK_FAIL) {
			return resource.NonRetryableError(fmt.Errorf("sqlserver configDatabaseCT status is fail"))
		} else {
			e = fmt.Errorf("sqlserver configDatabaseCT status illegal")
			return resource.NonRetryableError(e)
		}
	})

	if err != nil {
		log.Printf("[CRITAL]%s create sqlserver configDatabaseCT failed, reason:%+v", logId, err)
		return err
	}

	return resourceTencentCloudSqlserverConfigDatabaseCTRead(d, meta)
}

func resourceTencentCloudSqlserverConfigDatabaseCTDelete(d *schema.ResourceData, meta interface{}) error {
	defer logElapsed("resource.tencentcloud_sqlserver_config_database_c_t.delete")()
	defer inconsistentCheck(d, meta)()

	return nil
}

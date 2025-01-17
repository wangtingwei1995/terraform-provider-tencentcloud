/*
Provides a resource to create a vpc flow_log_config

Example Usage

If disable FlowLogs

```hcl
data "tencentcloud_availability_zones" "zones" {}

data "tencentcloud_images" "image" {
  image_type       = ["PUBLIC_IMAGE"]
  image_name_regex = "Final"
}

data "tencentcloud_instance_types" "instance_types" {
  filter {
    name   = "zone"
    values = [data.tencentcloud_availability_zones.zones.zones.0.name]
  }

  filter {
    name   = "instance-family"
    values = ["S5"]
  }

  cpu_core_count   = 2
  exclude_sold_out = true
}

resource "tencentcloud_cls_logset" "logset" {
  logset_name = "delogsetmo"
  tags        = {
    "test" = "test"
  }
}

resource "tencentcloud_cls_topic" "topic" {
  topic_name           = "topic"
  logset_id            = tencentcloud_cls_logset.logset.id
  auto_split           = false
  max_split_partitions = 20
  partition_count      = 1
  period               = 10
  storage_type         = "hot"
  tags                 = {
    "test" = "test",
  }
}

resource "tencentcloud_vpc" "vpc" {
  name       = "vpc-flow-log-vpc"
  cidr_block = "10.0.0.0/16"
}

resource "tencentcloud_subnet" "subnet" {
  availability_zone = data.tencentcloud_availability_zones.zones.zones.0.name
  name              = "vpc-flow-log-subnet"
  vpc_id            = tencentcloud_vpc.vpc.id
  cidr_block        = "10.0.0.0/16"
  is_multicast      = false
}

resource "tencentcloud_eni" "example" {
  name        = "vpc-flow-log-eni"
  vpc_id      = tencentcloud_vpc.vpc.id
  subnet_id   = tencentcloud_subnet.subnet.id
  description = "eni desc"
  ipv4_count  = 1
}

resource "tencentcloud_instance" "example" {
  instance_name            = "ci-test-eni-attach"
  availability_zone        = data.tencentcloud_availability_zones.zones.zones.0.name
  image_id                 = data.tencentcloud_images.image.images.0.image_id
  instance_type            = data.tencentcloud_instance_types.instance_types.instance_types.0.instance_type
  system_disk_type         = "CLOUD_PREMIUM"
  disable_security_service = true
  disable_monitor_service  = true
  vpc_id                   = tencentcloud_vpc.vpc.id
  subnet_id                = tencentcloud_subnet.subnet.id
}

resource "tencentcloud_eni_attachment" "example" {
  eni_id      = tencentcloud_eni.example.id
  instance_id = tencentcloud_instance.example.id
}

resource "tencentcloud_vpc_flow_log" "example" {
  flow_log_name        = "tf-example-vpc-flow-log"
  resource_type        = "NETWORKINTERFACE"
  resource_id          = tencentcloud_eni_attachment.example.eni_id
  traffic_type         = "ACCEPT"
  vpc_id               = tencentcloud_vpc.vpc.id
  flow_log_description = "this is a testing flow log"
  cloud_log_id         = tencentcloud_cls_topic.topic.id
  storage_type         = "cls"
  tags                 = {
    "testKey" = "testValue"
  }
}

resource "tencentcloud_vpc_flow_log_config" "config" {
  flow_log_id = tencentcloud_vpc_flow_log.example.id
  enable      = false
}
```

If enable FlowLogs

```hcl
resource "tencentcloud_vpc_flow_log_config" "config" {
  flow_log_id = tencentcloud_vpc_flow_log.example.id
  enable      = true
}
```

Import

vpc flow_log_config can be imported using the id, e.g.

```
terraform import tencentcloud_vpc_flow_log_config.flow_log_config flow_log_id
```
*/
package tencentcloud

import (
	"context"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

func resourceTencentCloudVpcFlowLogConfig() *schema.Resource {
	return &schema.Resource{
		Create: resourceTencentCloudVpcFlowLogConfigCreate,
		Read:   resourceTencentCloudVpcFlowLogConfigRead,
		Update: resourceTencentCloudVpcFlowLogConfigUpdate,
		Delete: resourceTencentCloudVpcFlowLogConfigDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"flow_log_id": {
				Required:    true,
				Type:        schema.TypeString,
				Description: "Flow log ID.",
			},

			"enable": {
				Required:    true,
				Type:        schema.TypeBool,
				Description: "If enable snapshot policy.",
			},
		},
	}
}

func resourceTencentCloudVpcFlowLogConfigCreate(d *schema.ResourceData, meta interface{}) error {
	defer logElapsed("resource.tencentcloud_vpc_flow_log_config.create")()
	defer inconsistentCheck(d, meta)()

	flowLogId := d.Get("flow_log_id").(string)

	d.SetId(flowLogId)

	return resourceTencentCloudVpcFlowLogConfigUpdate(d, meta)
}

func resourceTencentCloudVpcFlowLogConfigRead(d *schema.ResourceData, meta interface{}) error {
	defer logElapsed("resource.tencentcloud_vpc_flow_log_config.read")()
	defer inconsistentCheck(d, meta)()

	logId := getLogId(contextNil)

	ctx := context.WithValue(context.TODO(), logIdKey, logId)

	service := VpcService{client: meta.(*TencentCloudClient).apiV3Conn}

	flowLogId := d.Id()

	request := vpc.NewDescribeFlowLogsRequest()
	request.FlowLogId = &flowLogId

	flowLogs, err := service.DescribeFlowLogs(ctx, request)
	if err != nil {
		return err
	}

	if len(flowLogs) < 1 {
		d.SetId("")
		log.Printf("[WARN]%s resource `VpcFlowLogConfig` [%s] not found, please check if it has been deleted.\n", logId, d.Id())
		return nil
	}

	_ = d.Set("flow_log_id", flowLogId)

	flowLogConfig := flowLogs[0]

	if flowLogConfig.Enable != nil {
		_ = d.Set("enable", flowLogConfig.Enable)
	}

	return nil
}

func resourceTencentCloudVpcFlowLogConfigUpdate(d *schema.ResourceData, meta interface{}) error {
	defer logElapsed("resource.tencentcloud_vpc_flow_log_config.update")()
	defer inconsistentCheck(d, meta)()

	logId := getLogId(contextNil)

	var (
		enable         bool
		enableRequest  = vpc.NewEnableFlowLogsRequest()
		disableRequest = vpc.NewDisableFlowLogsRequest()
	)

	flowLogId := d.Id()

	if v, ok := d.GetOkExists("enable"); ok {
		enable = v.(bool)
	}

	if enable {
		enableRequest.FlowLogIds = []*string{&flowLogId}
		err := resource.Retry(writeRetryTimeout, func() *resource.RetryError {
			result, e := meta.(*TencentCloudClient).apiV3Conn.UseVpcClient().EnableFlowLogs(enableRequest)
			if e != nil {
				return retryError(e)
			} else {
				log.Printf("[DEBUG]%s api[%s] success, request body [%s], response body [%s]\n", logId, enableRequest.GetAction(), enableRequest.ToJsonString(), result.ToJsonString())
			}
			return nil
		})
		if err != nil {
			log.Printf("[CRITAL]%s update vpc flowLogConfig failed, reason:%+v", logId, err)
			return err
		}
	} else {
		disableRequest.FlowLogIds = []*string{&flowLogId}
		err := resource.Retry(writeRetryTimeout, func() *resource.RetryError {
			result, e := meta.(*TencentCloudClient).apiV3Conn.UseVpcClient().DisableFlowLogs(disableRequest)
			if e != nil {
				return retryError(e)
			} else {
				log.Printf("[DEBUG]%s api[%s] success, request body [%s], response body [%s]\n", logId, disableRequest.GetAction(), disableRequest.ToJsonString(), result.ToJsonString())
			}
			return nil
		})
		if err != nil {
			log.Printf("[CRITAL]%s update vpc flowLogConfig failed, reason:%+v", logId, err)
			return err
		}
	}

	return resourceTencentCloudVpcFlowLogConfigRead(d, meta)
}

func resourceTencentCloudVpcFlowLogConfigDelete(d *schema.ResourceData, meta interface{}) error {
	defer logElapsed("resource.tencentcloud_vpc_flow_log_config.delete")()
	defer inconsistentCheck(d, meta)()

	return nil
}

/*
Use this data source to query detailed information of waf ports

Example Usage

```hcl
data "tencentcloud_waf_ports" "example" {}
```

Or

```hcl
data "tencentcloud_waf_ports" "example" {
  edition     = "clb-waf"
  instance_id = "waf_2kxtlbky00b2v1fn"
}
```
*/
package tencentcloud

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	waf "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/waf/v20180125"
	"github.com/tencentcloudstack/terraform-provider-tencentcloud/tencentcloud/internal/helper"
)

func dataSourceTencentCloudWafPorts() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceTencentCloudWafPortsRead,
		Schema: map[string]*schema.Schema{
			"edition": {
				Optional:    true,
				Type:        schema.TypeString,
				Description: "Instance type, sparta-waf represents SAAS WAF, clb-waf represents CLB WAF.",
			},
			"instance_id": {
				Optional:    true,
				Type:        schema.TypeString,
				Description: "Instance unique ID.",
			},
			"http_ports": {
				Computed:    true,
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Http port list for instance.",
			},
			"https_ports": {
				Computed:    true,
				Type:        schema.TypeSet,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Https port list for instance.",
			},
			"result_output_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Used to save results.",
			},
		},
	}
}

func dataSourceTencentCloudWafPortsRead(d *schema.ResourceData, meta interface{}) error {
	defer logElapsed("data_source.tencentcloud_waf_ports.read")()
	defer inconsistentCheck(d, meta)()

	var (
		logId   = getLogId(contextNil)
		ctx     = context.WithValue(context.TODO(), logIdKey, logId)
		service = WafService{client: meta.(*TencentCloudClient).apiV3Conn}
		ports   *waf.DescribePortsResponseParams
	)

	paramMap := make(map[string]interface{})
	if v, ok := d.GetOk("edition"); ok {
		paramMap["Edition"] = helper.String(v.(string))
	}

	if v, ok := d.GetOk("instance_id"); ok {
		paramMap["InstanceID"] = helper.String(v.(string))
	}

	err := resource.Retry(readRetryTimeout, func() *resource.RetryError {
		result, e := service.DescribeWafPortsByFilter(ctx, paramMap)
		if e != nil {
			return retryError(e)
		}

		ports = result
		return nil
	})

	if err != nil {
		return err
	}

	if ports.HttpPorts != nil {
		_ = d.Set("http_ports", ports.HttpPorts)
	}

	if ports.HttpsPorts != nil {
		_ = d.Set("https_ports", ports.HttpsPorts)
	}

	d.SetId(*ports.RequestId)
	output, ok := d.GetOk("result_output_file")
	if ok && output.(string) != "" {
		if e := writeToFile(output.(string), d); e != nil {
			return e
		}
	}

	return nil
}

package tencentcloud

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccTencentCloudScfTriggersDataSource_basic(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccScfTriggersDataSource,
				Check:  resource.ComposeTestCheckFunc(testAccCheckTencentCloudDataSourceID("data.tencentcloud_scf_triggers.triggers")),
			},
		},
	})
}

const testAccScfTriggersDataSource = `

data "tencentcloud_scf_triggers" "triggers" {
  function_name = "keep-1676351130"
  namespace     = "default"
  order_by      = "add_time"
  order         = "DESC"
}

`

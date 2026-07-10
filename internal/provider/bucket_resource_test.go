package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

func TestAccBucketResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccBucketResourceWithRetentionConfig("test", "test bucket", "0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb_bucket.test", "name", "test"),
					resource.TestCheckResourceAttr("influxdb_bucket.test", "description", "test bucket"),
					resource.TestCheckResourceAttr("influxdb_bucket.test", "retention_period", "0"),
				),
			},
			// ImportState testing
			{
				ResourceName: "influxdb_bucket.test",
				ImportState:  true,
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccBucketResourceConfig("test-bucket", "test-bucket"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb_bucket.test", "name", "test-bucket"),
					resource.TestCheckResourceAttr("influxdb_bucket.test", "description", "test-bucket"),
					resource.TestCheckResourceAttr("influxdb_bucket.test", "retention_period", "2592000"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// TestAccBucketResourceOutOfBandDeletion verifies that a bucket deleted
// outside of Terraform is removed from state on refresh and planned for
// re-creation instead of failing the refresh.
func TestAccBucketResourceOutOfBandDeletion(t *testing.T) {
	if os.Getenv("INFLUXDB_TOKEN") == "" {
		t.Skip("INFLUXDB_TOKEN must be set: this test deletes the bucket out-of-band through the API")
	}

	bucketName := acctest.RandomWithPrefix("tf-bucket-test")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccBucketResourceConfig(bucketName, "out-of-band deletion test"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb_bucket.test", "name", bucketName),
				),
			},
			// Delete the bucket behind Terraform's back, then refresh: the
			// resource must drop out of state and the follow-up plan must be
			// a re-create, not an error.
			{
				PreConfig: func() {
					client := influxdb2.NewClient(os.Getenv("INFLUXDB_URL"), os.Getenv("INFLUXDB_TOKEN"))
					defer client.Close()

					ctx := context.Background()
					bucket, err := client.BucketsAPI().FindBucketByName(ctx, bucketName)
					if err != nil {
						t.Fatalf("finding bucket for out-of-band deletion: %s", err)
					}
					if err := client.BucketsAPI().DeleteBucket(ctx, bucket); err != nil {
						t.Fatalf("deleting bucket out-of-band: %s", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccBucketResourceWithRetentionConfig(name string, description string, retention_period string) string {
	return fmt.Sprintf(`
resource "influxdb_bucket" "test" {
  name = %[1]q
  description = %[2]q
  retention_period = %[3]q
  org_id = "`+os.Getenv("INFLUXDB_ORG_ID")+`"
}
`, name, description, retention_period)
}

func testAccBucketResourceConfig(name string, description string) string {
	return fmt.Sprintf(`
resource "influxdb_bucket" "test" {
  name = %[1]q
  description = %[2]q
  org_id = "`+os.Getenv("INFLUXDB_ORG_ID")+`"
}
`, name, description)
}

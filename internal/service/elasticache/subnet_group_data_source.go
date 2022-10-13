package elasticache

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
)

func DataSourceSubnetGroup() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceSubnetGroupRead,

		Schema: map[string]*schema.Schema{
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"subnet_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tftags.TagsSchema(),
		},
	}
}

func dataSourceSubnetGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*conns.AWSClient).ElastiCacheConn
	ignoreTagsConfig := meta.(*conns.AWSClient).IgnoreTagsConfig

	name := d.Get("name").(string)

	group, err := FindCacheSubnetGroupByName(conn, name)

	if err != nil {
		return fmt.Errorf("error finding ElastiCache Subnet Group (%s): %w", group, err)
	}

	d.SetId(aws.StringValue(group.CacheSubnetGroupName))

	var subnetIds []*string
	for _, subnet := range group.Subnets {
		subnetIds = append(subnetIds, subnet.SubnetIdentifier)
	}

	d.Set("arn", group.ARN)
	d.Set("description", group.CacheSubnetGroupDescription)
	d.Set("subnet_ids", flex.FlattenStringSet(subnetIds))
	d.Set("name", group.CacheSubnetGroupName)

	tags, err := ListTags(conn, d.Get("arn").(string))

	if err != nil {
		return fmt.Errorf("error listing tags for ElastiCache Subnet Group (%s): %w", group, err)
	}

	if err := d.Set("tags", tags.IgnoreAWS().IgnoreConfig(ignoreTagsConfig).Map()); err != nil {
		return fmt.Errorf("error setting tags: (%s)", err)
	}

	return nil
}

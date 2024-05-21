// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package mq_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/mq"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	tfmq "github.com/hashicorp/terraform-provider-aws/internal/service/mq"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func init() {
	acctest.RegisterServiceErrorCheckFunc(names.MQEndpointID, testAccErrorCheckSkip)
}

func testAccErrorCheckSkip(t *testing.T) resource.ErrorCheckFunc {
	return acctest.ErrorCheckSkipMessagesContaining(t,
		"To be determined...",
	)
}

func TestAccUser_serial(t *testing.T) {
	testCases := map[string]func(t *testing.T){
		"basic": testAccUser_basic,
		// "disappears": testAccAccountRegistration_disappears,
		// "kms key":    testAccAccountRegistration_optionalKMSKey,
	}

	acctest.RunSerialTests1Level(t, testCases, 0)
}

func testAccUser_basic(t *testing.T) {
	ctx := acctest.Context(t)
	resourceName := "aws_mq_user.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.MQEndpointID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		// CheckDestroy:             testAccCheckAccountRegistrationDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccConfigUser_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckUserExists(ctx, resourceName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportStateIdFunc: testAccUserImportStateIDFunc(ctx, resourceName),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccUserImportStateIDFunc(ctx context.Context, resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		err := testAccCheckUserExists(ctx, resourceName)(s)
		if err != nil {
			return "", fmt.Errorf("cannot generate import ID because resource doesn't exist: %v", err)
		}

		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", create.Error(names.MQ, create.ErrActionCheckingExistence, tfmq.ResourceNameUser, resourceName, errors.New("not found"))
		}

		brokerID := rs.Primary.Attributes["broker_id"]
		username := rs.Primary.ID
		return fmt.Sprintf("%s/%s", brokerID, username), nil
	}
}

func testAccCheckUserExists(ctx context.Context, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return create.Error(names.MQ, create.ErrActionCheckingExistence, tfmq.ResourceNameUser, resourceName, errors.New("not found"))
		}

		if rs.Primary.ID == "" {
			return create.Error(names.MQ, create.ErrActionCheckingExistence, tfmq.ResourceNameUser, resourceName, errors.New("not set"))
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).MQClient(ctx)
		brokerID := rs.Primary.Attributes["broker_id"]
		out, err := conn.DescribeUser(ctx, &mq.DescribeUserInput{
			BrokerId: &brokerID,
			Username: &rs.Primary.ID,
		})
		if err != nil {
			return create.Error(names.MQ, create.ErrActionCheckingExistence, tfmq.ResourceNameUser, rs.Primary.ID, err)
		}

		if out == nil {
			return create.Error(names.MQ, create.ErrActionCheckingExistence, tfmq.ResourceNameUser, rs.Primary.ID, errors.New("mq user not active"))
		}

		return nil
	}
}

func testAccConfigUser_basic() string {
	return `
resource "aws_mq_user" "test" {
	broker_id = "<WRITE_YOUR_BROKER_ID_HERE>"
	username = "testuser"
	password = "v98jxV3U0288"
}
`
}

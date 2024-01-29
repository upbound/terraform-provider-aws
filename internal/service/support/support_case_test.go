package support_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/support"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	tfsupport "github.com/hashicorp/terraform-provider-aws/internal/service/support"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func init() {
	acctest.RegisterServiceErrorCheckFunc(names.SupportEndpointID, testAccErrorCheckSkip)
}

func testAccErrorCheckSkip(t *testing.T) resource.ErrorCheckFunc {
	return acctest.ErrorCheckSkipMessagesContaining(t,
		"To be determined...",
	)
}

func TestAccSupportCase_serial(t *testing.T) {
	testCases := map[string]func(t *testing.T){
		"basic": testAccSupportCase_basic,
		// "disappears": testAccAccountRegistration_disappears,
		// "kms key":    testAccAccountRegistration_optionalKMSKey,
	}

	acctest.RunSerialTests1Level(t, testCases, 0)
}

func testAccSupportCase_basic(t *testing.T) {
	ctx := acctest.Context(t)
	resourceName := "aws_support_support_case.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			acctest.PreCheckRegion(t, endpoints.UsEast1RegionID, endpoints.UsWest2RegionID, endpoints.EuWest1RegionID)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.SupportEndpointID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		// CheckDestroy:             testAccCheckAccountRegistrationDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccConfigSupportCase_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSupportCaseIsActive(ctx, resourceName),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// testAccCheckAccountRegisterationIsActive verifies AuditManager is active in the current account/region combination
func testAccCheckSupportCaseIsActive(ctx context.Context, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return create.Error(names.Support, create.ErrActionCheckingExistence, tfsupport.ResourceNameSupportCase, name, errors.New("not found"))
		}

		if rs.Primary.ID == "" {
			return create.Error(names.Support, create.ErrActionCheckingExistence, tfsupport.ResourceNameSupportCase, name, errors.New("not set"))
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).SupportClient(ctx)
		out, err := conn.DescribeCases(ctx, &support.DescribeCasesInput{
			CaseIdList: []string{rs.Primary.ID},
		})
		if err != nil {
			return create.Error(names.Support, create.ErrActionCheckingExistence, tfsupport.ResourceNameSupportCase, rs.Primary.ID, err)
		}

		// TODO(cem): Consider performing extra checks here, like resource ID, or whatever checks there could be.
		if out == nil || len(out.Cases) != 1 {
			return create.Error(names.Support, create.ErrActionCheckingExistence, tfsupport.ResourceNameSupportCase, rs.Primary.ID, errors.New("support case not active"))
		}

		return nil
	}
}

func testAccConfigSupportCase_basic() string {
	return `
resource "aws_support_support_case" "test" {
	subject = "TEST CASE-Please ignore"
	communication_body = "This support case is created for AWS SDK development purposes."
	issue_type = "technical"
	language = "en"
	service_code = "support-api"
	category_code = "other"
	severity_code = "low"
}
`
}

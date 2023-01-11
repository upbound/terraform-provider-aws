package auditmanager_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/auditmanager"
	"github.com/aws/aws-sdk-go-v2/service/auditmanager/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	tfauditmanager "github.com/hashicorp/terraform-provider-aws/internal/service/auditmanager"
	"github.com/hashicorp/terraform-provider-aws/names"
)

func TestAccAuditManagerAccountRegistration_serial(t *testing.T) {
	t.Parallel()

	testCases := map[string]func(t *testing.T){
		"basic":      testAccAccountRegistration_basic,
		"disappears": testAccAccountRegistration_disappears,
		"kms key":    testAccAccountRegistration_optionalKMSKey,
	}

	acctest.RunSerialTests1Level(t, testCases, 0)
}

func testAccAccountRegistration_basic(t *testing.T) {
	resourceName := "aws_auditmanager_account_registration.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckPartitionHasService(names.AuditManagerEndpointID, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.AuditManagerEndpointID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckAccountRegistrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountRegistrationConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccountRegisterationIsActive(resourceName),
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

func testAccAccountRegistration_disappears(t *testing.T) {
	if os.Getenv("AUDITMANAGER_DEREGISTER_ACCOUNT_ON_DESTROY") == "" {
		t.Skip("Environment variable AUDITMANAGER_DEREGISTER_ACCOUNT_ON_DESTROY is not set")
	}

	resourceName := "aws_auditmanager_account_registration.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckPartitionHasService(names.AuditManagerEndpointID, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.AuditManagerEndpointID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckAccountRegistrationDestroy,
		Steps: []resource.TestStep{
			{
				// deregister_on_destroy must be enabled for the disappears helper to disable
				// audit manager on destroy and trigger the non-empty plan after state refresh
				Config: testAccAccountRegistrationConfig_deregisterOnDestroy(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccountRegisterationIsActive(resourceName),
					acctest.CheckFrameworkResourceDisappears(acctest.Provider, tfauditmanager.ResourceAccountRegistration, resourceName),
				),
			},
			{
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func testAccAccountRegistration_optionalKMSKey(t *testing.T) {
	resourceName := "aws_auditmanager_account_registration.test"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(t)
			acctest.PreCheckPartitionHasService(names.AuditManagerEndpointID, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, names.AuditManagerEndpointID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckAccountRegistrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAccountRegistrationConfig_KMSKey(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccountRegisterationIsActive(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "kms_key"),
				),
			},
			{
				Config: testAccAccountRegistrationConfig_basic(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccountRegisterationIsActive(resourceName),
					resource.TestCheckNoResourceAttr(resourceName, "kms_key"),
				),
			},
			{
				Config: testAccAccountRegistrationConfig_KMSKey(),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckAccountRegisterationIsActive(resourceName),
					resource.TestCheckResourceAttrSet(resourceName, "kms_key"),
				),
			},
		},
	})
}

// testAccCheckAccountRegistrationDestroy verfies GetAccountStatus does not return an error
//
// Since this resource manages activation/deactivation of AuditManager, there is nothing
// to destroy. Additionally, because registration may remain active depending on whether
// the deactivate_on_destroy attribute was set, this function does not check that account
// registration is inactive, simply that the status check returns a valid response.
func testAccCheckAccountRegistrationDestroy(s *terraform.State) error {
	ctx := context.Background()
	conn := acctest.Provider.Meta().(*conns.AWSClient).AuditManagerClient()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_auditmanager_account_registration" {
			continue
		}

		_, err := conn.GetAccountStatus(ctx, &auditmanager.GetAccountStatusInput{})
		if err != nil {
			return err
		}
	}

	return nil
}

// testAccCheckAccountRegisterationIsActive verifies AuditManager is active in the current account/region combination
func testAccCheckAccountRegisterationIsActive(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return create.Error(names.AuditManager, create.ErrActionCheckingExistence, tfauditmanager.ResNameAccountRegistration, name, errors.New("not found"))
		}

		if rs.Primary.ID == "" {
			return create.Error(names.AuditManager, create.ErrActionCheckingExistence, tfauditmanager.ResNameAccountRegistration, name, errors.New("not set"))
		}

		ctx := context.Background()
		conn := acctest.Provider.Meta().(*conns.AWSClient).AuditManagerClient()
		out, err := conn.GetAccountStatus(ctx, &auditmanager.GetAccountStatusInput{})
		if err != nil {
			return create.Error(names.AuditManager, create.ErrActionCheckingExistence, tfauditmanager.ResNameAccountRegistration, rs.Primary.ID, err)
		}
		if out == nil || out.Status != types.AccountStatusActive {
			return create.Error(names.AuditManager, create.ErrActionCheckingExistence, tfauditmanager.ResNameAccountRegistration, rs.Primary.ID, errors.New("audit manager not active"))
		}

		return nil
	}
}

func testAccAccountRegistrationConfig_basic() string {
	return `
resource "aws_auditmanager_account_registration" "test" {}
`
}

func testAccAccountRegistrationConfig_deregisterOnDestroy() string {
	return `
resource "aws_auditmanager_account_registration" "test" {
  deregister_on_destroy = true
}
`
}

func testAccAccountRegistrationConfig_KMSKey() string {
	return `
resource "aws_kms_key" "test" {}

resource "aws_auditmanager_account_registration" "test" {
  kms_key = aws_kms_key.test.arn
}
`
}

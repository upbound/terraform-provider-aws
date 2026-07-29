// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package backup_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/retry"
	tfbackup "github.com/hashicorp/terraform-provider-aws/internal/service/backup"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// Vault Lock retention values must be updatable in place: they are mutable via
// PutBackupVaultLockConfiguration before the lock date, and marking them
// ForceNew sends declarative callers (e.g. upjet, which refuses replacements)
// into a permanent update loop.
func TestVaultLockConfigurationUpdateSchema(t *testing.T) {
	t.Parallel()

	r := tfbackup.ResourceVaultLockConfiguration()

	if r.UpdateWithoutTimeout == nil {
		t.Error("resource must define UpdateWithoutTimeout")
	}

	type want struct {
		ForceNew bool
		Computed bool
	}
	cases := map[string]struct {
		args string
		want want
	}{
		"backup_vault_name is the identity and stays ForceNew": {args: "backup_vault_name", want: want{ForceNew: true}},
		"changeable_for_days updatable":                        {args: "changeable_for_days"},
		"max_retention_days updatable":                         {args: "max_retention_days"},
		"min_retention_days updatable":                         {args: "min_retention_days"},
		"lock_date exposed as computed":                        {args: "lock_date", want: want{Computed: true}},
		"locked exposed as computed":                           {args: "locked", want: want{Computed: true}},
	}

	schemaMap := r.SchemaMap()
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			s, ok := schemaMap[tc.args]
			if !ok {
				t.Fatalf("attribute %q not found in schema", tc.args)
			}

			got := want{ForceNew: s.ForceNew, Computed: s.Computed}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("attribute %q (-want +got):\n%s", tc.args, diff)
			}
		})
	}
}

// AWS bounds ChangeableForDays to [3, 36500]; out-of-range values must fail at
// plan time, not at apply time.
func TestVaultLockConfigurationChangeableForDaysValidation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		args int
		want bool
	}{
		"below minimum": {args: 2, want: true},
		"minimum":       {args: 3},
		"maximum":       {args: 36500},
		"above maximum": {args: 36501, want: true},
	}

	validate := tfbackup.ResourceVaultLockConfiguration().SchemaMap()["changeable_for_days"].ValidateFunc
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, errs := validate(tc.args, "changeable_for_days")
			if got := len(errs) > 0; got != tc.want {
				t.Errorf("validate(%d): wantErr = %t, got errors %v", tc.args, tc.want, errs)
			}
		})
	}
}

func TestAccBackupVaultLockConfiguration_basic(t *testing.T) {
	ctx := acctest.Context(t)
	var vault backup.DescribeBackupVaultOutput

	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_backup_vault_lock_configuration.test"
	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.BackupServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckVaultLockConfigurationDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVaultLockConfigurationConfig_all(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultLockConfigurationExists(ctx, t, resourceName, &vault),
					resource.TestCheckResourceAttr(resourceName, "changeable_for_days", "3"),
					resource.TestCheckResourceAttr(resourceName, "max_retention_days", "1200"),
					resource.TestCheckResourceAttr(resourceName, "min_retention_days", "7"),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// These are not returned by the API
				ImportStateVerifyIgnore: []string{"changeable_for_days"},
			},
		},
	})
}

func TestAccBackupVaultLockConfiguration_disappears(t *testing.T) {
	ctx := acctest.Context(t)
	var vault backup.DescribeBackupVaultOutput

	rName := acctest.RandomWithPrefix(t, acctest.ResourcePrefix)
	resourceName := "aws_backup_vault_lock_configuration.test"
	acctest.ParallelTest(ctx, t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(ctx, t); testAccPreCheck(ctx, t) },
		ErrorCheck:               acctest.ErrorCheck(t, names.BackupServiceID),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckVaultLockConfigurationDestroy(ctx, t),
		Steps: []resource.TestStep{
			{
				Config: testAccVaultLockConfigurationConfig_all(rName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckVaultLockConfigurationExists(ctx, t, resourceName, &vault),
					acctest.CheckSDKResourceDisappears(ctx, t, tfbackup.ResourceVaultLockConfiguration(), resourceName),
				),
				ExpectNonEmptyPlan: true,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
					},
					PostApplyPostRefresh: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceName, plancheck.ResourceActionCreate),
					},
				},
			},
		},
	})
}

func testAccCheckVaultLockConfigurationDestroy(ctx context.Context, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.ProviderMeta(ctx, t).BackupClient(ctx)
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_backup_vault_lock_configuration" {
				continue
			}

			_, err := tfbackup.FindBackupVaultByName(ctx, conn, rs.Primary.ID)

			if retry.NotFound(err) {
				continue
			}

			if err != nil {
				return err
			}

			return fmt.Errorf("Backup Vault Lock Configuration %s still exists", rs.Primary.ID)
		}

		return nil
	}
}

func testAccCheckVaultLockConfigurationExists(ctx context.Context, t *testing.T, n string, v *backup.DescribeBackupVaultOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		conn := acctest.ProviderMeta(ctx, t).BackupClient(ctx)

		output, err := tfbackup.FindBackupVaultByName(ctx, conn, rs.Primary.ID)

		if err != nil {
			return err
		}

		*v = *output

		return nil
	}
}

func testAccVaultLockConfigurationConfig_all(rName string) string {
	return fmt.Sprintf(`
resource "aws_backup_vault" "test" {
  name = %[1]q
}

resource "aws_backup_vault_lock_configuration" "test" {
  backup_vault_name   = aws_backup_vault.test.name
  changeable_for_days = 3
  max_retention_days  = 1200
  min_retention_days  = 7
}
`, rName)
}

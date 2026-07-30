---
subcategory: "Backup"
layout: "aws"
page_title: "AWS: aws_backup_vault_lock_configuration"
description: |-
  Provides an AWS Backup vault lock configuration resource.
---

# Resource: aws_backup_vault_lock_configuration

Provides an AWS Backup vault lock configuration resource.

~> **Note:** Updates replace the whole lock configuration via [`PutBackupVaultLockConfiguration`](https://docs.aws.amazon.com/aws-backup/latest/devguide/API_PutBackupVaultLockConfiguration.html) and are only possible while the vault lock is changeable: always in `governance` mode, and until the lock date in `compliance` mode. On and after `lock_date` the vault lock is immutable — AWS permanently rejects any update or deletion of the lock configuration. AWS computes the lock date from the time of the call, so updating a `compliance` mode configuration during its grace period may move `lock_date` forward.

## Example Usage

```terraform
resource "aws_backup_vault_lock_configuration" "test" {
  backup_vault_name   = "example_backup_vault"
  changeable_for_days = 3
  max_retention_days  = 1200
  min_retention_days  = 7
}
```

## Argument Reference

This resource supports the following arguments:

* `region` - (Optional) Region where this resource will be [managed](https://docs.aws.amazon.com/general/latest/gr/rande.html#regional-endpoints). Defaults to the Region set in the [provider configuration](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#aws-configuration-reference).
* `backup_vault_name` - (Required) Name of the backup vault to add a lock configuration for.
* `changeable_for_days` - (Optional) The number of days before the lock date, between `3` and `36500`. If omitted creates a vault lock in `governance` mode, otherwise it will create a vault lock in `compliance` mode.
* `max_retention_days` - (Optional) The maximum retention period that the vault retains its recovery points.
* `min_retention_days` - (Optional) The minimum retention period that the vault retains its recovery points.

## Attribute Reference

This resource exports the following attributes in addition to the arguments above:

* `backup_vault_name` - The name of the vault.
* `backup_vault_arn` - The ARN of the vault.
* `locked` - Whether Vault Lock is currently protecting the backup vault.
* `lock_date` - Date and time, in RFC3339 format, when the vault lock becomes immutable. Not set in `governance` mode.

## Import

In Terraform v1.5.0 and later, use an [`import` block](https://developer.hashicorp.com/terraform/language/import) to import Backup vault lock configuration using the `name`. For example:

```terraform
import {
  to = aws_backup_vault_lock_configuration.test
  id = "TestVault"
}
```

Using `terraform import`, import Backup vault lock configuration using the `name`. For example:

```console
% terraform import aws_backup_vault_lock_configuration.test TestVault
```

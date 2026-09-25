---
subcategory: "Bedrock AgentCore"
layout: "aws"
page_title: "AWS: aws_bedrockagentcore_code_interpreter"
description: |-
  Manages an AWS Bedrock AgentCore Code Interpreter.
---

# Resource: aws_bedrockagentcore_code_interpreter

Manages an AWS Bedrock AgentCore Code Interpreter. Code Interpreter provides a secure environment for AI agents to execute Python code, enabling data analysis, calculations, and file processing capabilities.

## Example Usage

### Basic Usage

```terraform
resource "aws_bedrockagentcore_code_interpreter" "example" {
  name        = "example-code-interpreter"
  description = "Code interpreter for data analysis"

  network_configuration {
    network_mode = "PUBLIC"
  }
}
```

### Code Interpreter with Execution Role

```terraform
data "aws_iam_policy_document" "assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["bedrock-agentcore.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "example" {
  name               = "bedrock-agentcore-code-interpreter-role"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}

resource "aws_bedrockagentcore_code_interpreter" "example" {
  name               = "example-code-interpreter"
  description        = "Code interpreter with custom execution role"
  execution_role_arn = aws_iam_role.example.arn

  network_configuration {
    network_mode = "SANDBOX"
  }
}
```

### Code Interpreter with an Amazon S3 Files Mount

```terraform
resource "aws_bedrockagentcore_code_interpreter" "example" {
  name               = "example-code-interpreter"
  description        = "Code interpreter with an S3 Files mount"
  execution_role_arn = aws_iam_role.example.arn

  network_configuration {
    network_mode = "VPC"

    vpc_config {
      security_groups = [aws_security_group.example.id]
      subnets         = aws_subnet.example[*].id
    }
  }

  filesystem_configuration {
    s3_files_configuration {
      access_point_arn = aws_s3files_access_point.example.arn
      file_system_arn  = aws_s3files_file_system.example.arn
      mount_path       = "/mnt/s3data"
    }
  }
}
```

## Argument Reference

The following arguments are required:

* `name` - (Required) Name of the code interpreter.
* `network_configuration` - (Required) Network configuration for the code interpreter. See [`network_configuration`](#network_configuration) below.

The following arguments are optional:

* `region` - (Optional) Region where this resource will be [managed](https://docs.aws.amazon.com/general/latest/gr/rande.html#regional-endpoints). Defaults to the Region set in the [provider configuration](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#aws-configuration-reference).
* `certificate` - (Optional) Certificates to install in the code interpreter. Between 1 and 200 blocks are supported. See [`certificate`](#certificate) below.
* `description` - (Optional) Description of the code interpreter.
* `execution_role_arn` - (Optional) ARN of the IAM role that the code interpreter assumes for execution. Required when using `SANDBOX` network mode.
* `filesystem_configuration` - (Optional) List of filesystems to mount into every session started from the code interpreter. Up to 4 entries are supported, of which at most 2 can be Amazon S3 Files access points and at most 2 can be Amazon EFS access points. Requires `VPC` network mode. See [`filesystem_configuration`](#filesystem_configuration) below.
* `client_token` - (Optional) Unique identifier for request idempotency. If not provided, one will be generated automatically.
* `tags` - (Optional) Key-value map of resource tags. If configured with a provider [`default_tags` configuration block](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#default_tags-configuration-block) present, tags with matching keys will overwrite those defined at the provider-level.

### `certificate`

The `certificate` block supports the following:

* `location` - (Required) Location from which to retrieve the certificate. See [`certificates.location`](#certificateslocation) below.

### `certificate.location`

The certificate `location` object supports the following:

* `secrets_manager` - (Required) AWS Secrets Manager location of the certificate. See [`secrets_manager`](#secrets_manager) below.

### `secrets_manager`

The `secrets_manager` object supports the following:

* `secret_arn` - (Required) ARN of the AWS Secrets Manager secret containing the certificate.

### `filesystem_configuration`

Each `filesystem_configuration` block describes a single filesystem to mount into sessions started from the code interpreter. The list can contain up to 4 entries. Each block must specify exactly one of `s3_files_configuration` or `efs_configuration`, and each mount path must be unique across the list.

Mounting a filesystem requires the code interpreter to use `VPC` network mode, the execution role to allow the corresponding mount actions, and the mount target security group to allow TCP port `2049` from the code interpreter security group.

* `s3_files_configuration` - (Optional) Amazon S3 Files access point to mount as shared file storage. Exactly one of `s3_files_configuration` or `efs_configuration` must be specified. See [`s3_files_configuration`](#s3_files_configuration) below.
* `efs_configuration` - (Optional) Amazon EFS access point to mount as shared file storage. Exactly one of `s3_files_configuration` or `efs_configuration` must be specified. See [`efs_configuration`](#efs_configuration) below.

### `s3_files_configuration`

The `s3_files_configuration` block supports the following:

* `access_point_arn` - (Required) ARN of the Amazon S3 Files access point to mount.
* `file_system_arn` - (Required) ARN of the Amazon S3 Files file system that owns the access point.
* `mount_path` - (Required) Absolute path within the session at which the access point is mounted. Must be under `/mnt` with exactly one subdirectory level (for example, `/mnt/s3data`).

### `efs_configuration`

The `efs_configuration` block supports the following:

* `access_point_arn` - (Required) ARN of the Amazon EFS access point to mount.
* `file_system_arn` - (Required) ARN of the Amazon EFS file system that owns the access point.
* `mount_path` - (Required) Absolute path within the session at which the access point is mounted. Must be under `/mnt` with exactly one subdirectory level (for example, `/mnt/efs`).

### `network_configuration`

The `network_configuration` object supports the following:

* `network_mode` - (Required) Network mode for the code interpreter. Valid values: `PUBLIC`, `SANDBOX`, `VPC`.
* `vpc_config` - (Optional) VPC configuration. See [`vpc_config`](#vpc_config) below.

### `vpc_config`

The `vpc_config` block supports the following:

* `security_groups` - (Required) Security groups associated with the VPC configuration.
* `subnets` - (Required) Subnets associated with the VPC configuration.

## Attribute Reference

This resource exports the following attributes in addition to the arguments above:

* `code_interpreter_arn` - ARN of the Code Interpreter.
* `code_interpreter_id` - Unique identifier of the Code Interpreter.
* `tags_all` - A map of tags assigned to the resource, including those inherited from the provider [`default_tags` configuration block](https://registry.terraform.io/providers/hashicorp/aws/latest/docs#default_tags-configuration-block).

## Timeouts

[Configuration options](https://developer.hashicorp.com/terraform/language/resources/syntax#operation-timeouts):

* `create` - (Default `30m`)
* `delete` - (Default `30m`)

## Import

In Terraform v1.5.0 and later, use an [`import` block](https://developer.hashicorp.com/terraform/language/import) to import Bedrock AgentCore Code Interpreter using the code interpreter ID. For example:

```terraform
import {
  to = aws_bedrockagentcore_code_interpreter.example
  id = "CODEINTERPRETER1234567890"
}
```

Using `terraform import`, import Bedrock AgentCore Code Interpreter using the code interpreter ID. For example:

```console
% terraform import aws_bedrockagentcore_code_interpreter.example CODEINTERPRETER1234567890
```

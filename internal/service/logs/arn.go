// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package logs

import (
	"strings"
)

const (
	logGroupARNWildcardSuffix = ":*"
)

// TrimLogGroupARNWildcardSuffix trims any wilcard suffix from a Log Group ARN.
func TrimLogGroupARNWildcardSuffix(arn string) string {
	return strings.TrimSuffix(arn, logGroupARNWildcardSuffix)
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package conns

import (
	session_sdkv1 "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/smithy-go/middleware"
)

// AppendAPIOptions appends the specified AWS client APIOptions to
// the client c.
func (c *AWSClient) AppendAPIOptions(options ...func(stack *middleware.Stack) error) {
	c.awsConfig.APIOptions = append(c.awsConfig.APIOptions, options...)
}

// Session returns the associated session with this client.
func (c *AWSClient) Session() *session_sdkv1.Session {
	return c.session
}

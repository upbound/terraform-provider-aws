// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package conns

import (
	"github.com/aws/smithy-go/middleware"
)

// AppendAPIOptions appends the specified AWS client APIOptions to
// the client c.
func (c *AWSClient) AppendAPIOptions(options ...func(stack *middleware.Stack) error) {
	c.awsConfig.APIOptions = append(c.awsConfig.APIOptions, options...)
}

// SetAccountID sets accountID of this client.
func (c *AWSClient) SetAccountID(accountID string) {
	c.accountID = accountID
}

// GetServicePackages returns the servicePackages map for backward compatibility.
func (c *AWSClient) GetServicePackages() map[string]ServicePackage {
	return c.servicePackages
}

// SetServicePackagesField sets the servicePackages field directly for backward compatibility.
func (c *AWSClient) SetServicePackagesField(servicePackages map[string]ServicePackage) {
	c.servicePackages = servicePackages
}

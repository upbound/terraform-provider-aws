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

// Session returns the associated session with this client.
// DEPRECATED: This method is deprecated as AWS SDK v1 session is no longer available.
// Use AppendAPIOptions for middleware functionality instead.
func (c *AWSClient) Session() any {
	// Return nil as session is not available in AWS SDK v2
	// Crossplane provider should migrate to use AppendAPIOptions for metrics collection
	return nil
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

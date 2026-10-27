// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package mq

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/mq"
	awstypes "github.com/aws/aws-sdk-go-v2/service/mq/types"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
)

func userSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	ctx := context.Background()
	r, err := newResourceUser(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var resp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	return resp
}

func TestUserSchemaServerDefaultedAttributesAreComputed(t *testing.T) {
	t.Parallel()

	s := userSchema(t).Schema
	if len(s.Blocks) != 0 {
		t.Errorf("schema declares blocks %v, want none: a block cannot be computed", s.Blocks)
	}
	cases := map[string]struct {
		wantOptional bool
	}{
		"console_access":   {wantOptional: true},
		"replication_user": {wantOptional: true},
		"pending":          {wantOptional: false},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			attr, ok := s.Attributes[name]
			if !ok {
				t.Fatalf("attribute %q missing", name)
			}
			got := struct{ Optional, Computed bool }{attr.IsOptional(), attr.IsComputed()}
			want := struct{ Optional, Computed bool }{c.wantOptional, true}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("-want +got:\n%s", diff)
			}
		})
	}
}

func TestUserSchemaHasNoNestedAttributes(t *testing.T) {
	t.Parallel()

	for name, a := range userSchema(t).Schema.Attributes {
		if _, ok := a.(schema.NestedAttribute); ok {
			t.Errorf("attribute %q is a nested attribute: protocol version 5 cannot serve it, so Terraform fails to load the provider schema", name)
		}
	}
}

var boolComparer = cmp.Comparer(func(a, b types.Bool) bool { return a.Equal(b) })

func TestUserRefreshFromOutputKeepsReplicationUserWhenAbsent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cases := map[string]struct {
		held types.Bool
		api  *bool
		want types.Bool
	}{
		"api value wins":            {held: types.BoolValue(false), api: aws.Bool(true), want: types.BoolValue(true)},
		"absent keeps configured":   {held: types.BoolValue(false), api: nil, want: types.BoolValue(false)},
		"absent keeps null":         {held: types.BoolNull(), api: nil, want: types.BoolNull()},
		"absent settles unknown":    {held: types.BoolUnknown(), api: nil, want: types.BoolNull()},
		"api value settles unknown": {held: types.BoolUnknown(), api: aws.Bool(false), want: types.BoolValue(false)},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			rd := resourceUserData{ReplicationUser: c.held}
			out := &mq.DescribeUserOutput{Username: aws.String("u"), ReplicationUser: c.api}
			if d := rd.refreshFromOutput(ctx, out); d.HasError() {
				t.Fatal(d)
			}
			if diff := cmp.Diff(c.want, rd.ReplicationUser, boolComparer); diff != "" {
				t.Errorf("-want +got:\n%s", diff)
			}
		})
	}
}

func TestUserResolveUnknownFromOutput(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	out := &mq.DescribeUserOutput{
		Username:        aws.String("u"),
		ConsoleAccess:   aws.Bool(false),
		ReplicationUser: aws.Bool(false),
		Pending: &awstypes.UserPendingChanges{
			PendingChange: awstypes.ChangeTypeUpdate,
			ConsoleAccess: aws.Bool(true),
			Groups:        []string{"app"},
		},
	}
	pendingFromAPI, d := fwtypes.NewObjectValueOf(ctx, &pendingModel{
		ConsoleAccess: types.BoolValue(true),
		PendingChange: types.StringValue("UPDATE"),
		Groups:        fwtypes.NewListValueOfMust[types.String](ctx, []attr.Value{types.StringValue("app")}),
	})
	if d.HasError() {
		t.Fatal(d)
	}
	pendingComparer := cmp.Comparer(func(a, b fwtypes.ObjectValueOf[pendingModel]) bool { return a.Equal(b) })

	type resolved struct {
		ConsoleAccess, ReplicationUser types.Bool
		Pending                        fwtypes.ObjectValueOf[pendingModel]
	}
	cases := map[string]struct {
		plan resourceUserData
		want resolved
	}{
		"unknown values settle from the api, known values stay planned": {
			plan: resourceUserData{ConsoleAccess: types.BoolUnknown(), ReplicationUser: types.BoolValue(true), Pending: fwtypes.NewObjectValueOfUnknown[pendingModel](ctx)},
			want: resolved{ConsoleAccess: types.BoolValue(false), ReplicationUser: types.BoolValue(true), Pending: pendingFromAPI},
		},
		"known pending stays planned": {
			plan: resourceUserData{ConsoleAccess: types.BoolValue(true), ReplicationUser: types.BoolUnknown(), Pending: fwtypes.NewObjectValueOfNull[pendingModel](ctx)},
			want: resolved{ConsoleAccess: types.BoolValue(true), ReplicationUser: types.BoolValue(false), Pending: fwtypes.NewObjectValueOfNull[pendingModel](ctx)},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			rd := c.plan
			if d := rd.resolveUnknownFromOutput(ctx, out); d.HasError() {
				t.Fatal(d)
			}
			got := resolved{ConsoleAccess: rd.ConsoleAccess, ReplicationUser: rd.ReplicationUser, Pending: rd.Pending}
			if diff := cmp.Diff(c.want, got, boolComparer, pendingComparer); diff != "" {
				t.Errorf("-want +got:\n%s", diff)
			}
		})
	}
}

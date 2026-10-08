// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: MPL-2.0

package dynamodb_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	tfdynamodb "github.com/hashicorp/terraform-provider-aws/internal/service/dynamodb"
)

// TestCustomDiffGlobalSecondaryIndexUpjetRawValues computes the diff of an
// existing table the way Upjet's SDKv2 external client does: RawState and
// RawPlan both hold the refreshed state, and RawConfig holds the desired
// configuration built from the managed resource's parameters.
func TestCustomDiffGlobalSecondaryIndexUpjetRawValues(t *testing.T) {
	t.Parallel()

	stateGSI := func(name, hashKey string, readCapacity int) map[string]any {
		return map[string]any{
			"name": name, "hash_key": hashKey, "projection_type": "ALL",
			"read_capacity": readCapacity, "write_capacity": 5,
			"key_schema": []any{map[string]any{"attribute_name": hashKey, "key_type": "HASH"}},
		}
	}
	hashKeyGSI := func(name, hashKey string, readCapacity int) map[string]any {
		return map[string]any{
			"name": name, "hash_key": hashKey, "projection_type": "ALL",
			"read_capacity": readCapacity, "write_capacity": 5,
		}
	}
	keySchemaGSI := func(name, hashKey string, readCapacity int) map[string]any {
		return map[string]any{
			"name": name, "projection_type": "ALL",
			"read_capacity": readCapacity, "write_capacity": 5,
			"key_schema": []any{map[string]any{"attribute_name": hashKey, "key_type": "HASH"}},
		}
	}
	attributes := func(names ...string) []any {
		attrs := make([]any, 0, len(names))
		for _, n := range names {
			attrs = append(attrs, map[string]any{"name": n, "type": "S"})
		}
		return attrs
	}
	table := func(gsis, attrs []any) map[string]any {
		return map[string]any{
			"id": "test", "name": "test", "hash_key": "pk", "billing_mode": "PROVISIONED",
			"read_capacity": 5, "write_capacity": 5,
			"attribute": attrs, "global_secondary_index": gsis,
		}
	}

	type args struct {
		config map[string]any
	}
	type want struct {
		gsiChanged bool
	}
	cases := map[string]struct {
		args args
		want want
	}{
		"UnchangedHashKeySyntax": {
			args: args{config: table([]any{hashKeyGSI("gsi1", "gsi1pk", 5), hashKeyGSI("gsi3", "gsi3pk", 5)}, attributes("pk", "gsi1pk", "gsi3pk"))},
			want: want{gsiChanged: false},
		},
		"UnchangedKeySchemaSyntax": {
			args: args{config: table([]any{keySchemaGSI("gsi1", "gsi1pk", 5), keySchemaGSI("gsi3", "gsi3pk", 5)}, attributes("pk", "gsi1pk", "gsi3pk"))},
			want: want{gsiChanged: false},
		},
		"ReadCapacityChanged": {
			args: args{config: table([]any{hashKeyGSI("gsi1", "gsi1pk", 10), hashKeyGSI("gsi3", "gsi3pk", 5)}, attributes("pk", "gsi1pk", "gsi3pk"))},
			want: want{gsiChanged: true},
		},
		"GSIAdded": {
			args: args{config: table([]any{hashKeyGSI("gsi1", "gsi1pk", 5), hashKeyGSI("gsi3", "gsi3pk", 5), hashKeyGSI("gsi2", "gsi2pk", 5)}, attributes("pk", "gsi1pk", "gsi3pk", "gsi2pk"))},
			want: want{gsiChanged: true},
		},
		"GSIRemoved": {
			args: args{config: table([]any{hashKeyGSI("gsi1", "gsi1pk", 5)}, attributes("pk", "gsi1pk"))},
			want: want{gsiChanged: true},
		},
		"GSIRenamed": {
			args: args{config: table([]any{hashKeyGSI("gsi1", "gsi1pk", 5), hashKeyGSI("gsi4", "gsi3pk", 5)}, attributes("pk", "gsi1pk", "gsi3pk"))},
			want: want{gsiChanged: true},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			r := tfdynamodb.ResourceTable()
			block := r.CoreConfigSchema()

			priorVal, err := schema.JSONMapToStateValue(table([]any{stateGSI("gsi1", "gsi1pk", 5), stateGSI("gsi3", "gsi3pk", 5)}, attributes("pk", "gsi1pk", "gsi3pk")), block)
			if err != nil {
				t.Fatal(err)
			}
			state, err := r.ShimInstanceStateFromValue(priorVal)
			if err != nil {
				t.Fatal(err)
			}
			stateVal, err := state.AttrsAsObjectValue(block.ImpliedType())
			if err != nil {
				t.Fatal(err)
			}
			configVal, err := schema.JSONMapToStateValue(tc.args.config, block)
			if err != nil {
				t.Fatal(err)
			}
			state.RawState = stateVal
			state.RawPlan = stateVal
			state.RawConfig = configVal

			diff, err := schema.InternalMap(r.SchemaMap()).Diff(ctx, state, terraform.NewResourceConfigRaw(tc.args.config), tfdynamodb.CustomDiffGlobalSecondaryIndex, nil, false)
			if err != nil {
				t.Fatal(err)
			}

			var got want
			if diff != nil {
				for k := range diff.Attributes {
					got.gsiChanged = got.gsiChanged || strings.HasPrefix(k, "global_secondary_index.")
				}
			}
			if d := cmp.Diff(tc.want, got, cmp.AllowUnexported(want{})); d != "" {
				t.Errorf("global_secondary_index diff: -want, +got:\n%s", d)
			}
		})
	}
}

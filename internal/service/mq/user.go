// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package mq

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/mq"
	awstypes "github.com/aws/aws-sdk-go-v2/service/mq/types"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	"github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// diagErrorFramework is a helper method that creates an error diagnostic.
// This method was formerly at internal/create/errors.go
func diagErrorFramework(service, action, resource, id string, gotError error) diag.Diagnostic {
	return diag.NewErrorDiagnostic(
		create.ProblemStandardMessage(service, action, resource, id, nil),
		gotError.Error(),
	)
}

// @FrameworkResource(name="User")
func newResourceUser(_ context.Context) (resource.ResourceWithConfigure, error) {
	return &resourceUser{}, nil
}

const (
	ResourceNameUser = "User"
)

type resourceUser struct {
	framework.ResourceWithConfigure
}

func (r *resourceUser) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "aws_mq_user"
}

// Schema returns the schema for this resource.
func (r *resourceUser) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"broker_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"console_access": schema.BoolAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.Bool{
					consoleAccessPlanModifier{},
				},
			},
			"groups": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
				PlanModifiers: []planmodifier.List{
					listSortModifier{},
				},
			},
			"id": framework.IDAttribute(),
			"password": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(12),
				},
			},
			"replication_user": schema.BoolAttribute{
				Optional: true,
			},
			"username": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			// Pending changes information returned by the AWS API. This is a
			// computed-only attribute that tracks pending modifications to
			// the user.
			"pending": schema.ObjectAttribute{
				CustomType: fwtypes.NewObjectTypeOf[pendingModel](ctx),
				Computed:   true,
				Optional:   false,
			},
		},
	}
}

func (r *resourceUser) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan resourceUserData
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().MQClient(ctx)

	input := &mq.CreateUserInput{
		BrokerId:        flex.StringFromFramework(ctx, plan.BrokerID),
		Password:        flex.StringFromFramework(ctx, plan.Password),
		Username:        flex.StringFromFramework(ctx, plan.Username),
		ConsoleAccess:   flex.BoolFromFramework(ctx, plan.ConsoleAccess),
		Groups:          flex.ExpandFrameworkStringValueList(ctx, plan.Groups),
		ReplicationUser: flex.BoolFromFramework(ctx, plan.ReplicationUser),
	}
	_, err := conn.CreateUser(ctx, input)
	if err != nil {
		response.Diagnostics.Append(diagErrorFramework(names.MQ, create.ErrActionCreating, ResourceNameUser, fmt.Sprintf("%s/%s", plan.BrokerID.ValueString(), plan.Username.ValueString()), err))
		return
	}

	// Create API call returns no data. Get resource details.
	userDetails, err := findUserByID(ctx, conn, plan.BrokerID.ValueString(), plan.Username.ValueString())
	if err != nil {
		response.Diagnostics.Append(diagErrorFramework(names.MQ, create.ErrActionCreating, ResourceNameUser, fmt.Sprintf("%s/%s", plan.BrokerID.ValueString(), plan.Username.ValueString()), err))
		return
	}

	state := plan

	response.Diagnostics.Append(state.refreshFromOutput(ctx, userDetails)...)
	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (r *resourceUser) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state resourceUserData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().MQClient(ctx)

	userDetails, err := findUserByID(ctx, conn, state.BrokerID.ValueString(), state.ID.ValueString())
	if tfresource.NotFound(err) {
		create.LogNotFoundRemoveState(names.MQ, create.ErrActionReading, ResourceNameUser, fmt.Sprintf("%s/%s", state.BrokerID.ValueString(), state.ID.ValueString()))
		response.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		response.Diagnostics.Append(diagErrorFramework(names.MQ, create.ErrActionReading, ResourceNameUser, fmt.Sprintf("%s/%s", state.BrokerID.ValueString(), state.ID.ValueString()), err))
		return
	}

	response.Diagnostics.Append(state.refreshFromOutput(ctx, userDetails)...)
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

func (r *resourceUser) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var state, plan resourceUserData

	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	if userHasChanges(plan, state) {
		conn := r.Meta().MQClient(ctx)

		input := &mq.UpdateUserInput{
			BrokerId:        flex.StringFromFramework(ctx, plan.BrokerID),
			Password:        flex.StringFromFramework(ctx, plan.Password),
			Username:        flex.StringFromFramework(ctx, plan.ID),
			ConsoleAccess:   flex.BoolFromFramework(ctx, plan.ConsoleAccess),
			Groups:          flex.ExpandFrameworkStringValueList(ctx, plan.Groups),
			ReplicationUser: flex.BoolFromFramework(ctx, plan.ReplicationUser),
		}
		_, err := conn.UpdateUser(ctx, input)
		if err != nil {
			response.Diagnostics.Append(diagErrorFramework(names.MQ, create.ErrActionUpdating, ResourceNameUser, fmt.Sprintf("%s/%s", state.BrokerID.ValueString(), state.ID.ValueString()), err))
			return
		}
	}

	response.Diagnostics.Append(response.State.Set(ctx, &plan)...)
}

func (r *resourceUser) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var state resourceUserData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().MQClient(ctx)

	input := &mq.DeleteUserInput{
		BrokerId: flex.StringFromFramework(ctx, state.BrokerID),
		Username: flex.StringFromFramework(ctx, state.ID),
	}
	_, err := conn.DeleteUser(ctx, input)
	if err != nil {
		response.Diagnostics.Append(diagErrorFramework(names.MQ, create.ErrActionDeleting, ResourceNameUser, fmt.Sprintf("%s/%s", state.BrokerID.ValueString(), state.ID.ValueString()), err))
		return
	}
}

func (r *resourceUser) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	parts := strings.Split(request.ID, "/")
	if len(parts) != 2 {
		response.Diagnostics.AddError("Resource Import Invalid ID", fmt.Sprintf("wrong format of import ID (%s), use: broker-id/username'", request.ID))
		return
	}

	brokerID := parts[0]
	username := parts[1]
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("id"), username)...)
	response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("broker_id"), brokerID)...)
}

func findUserByID(ctx context.Context, conn *mq.Client, brokerID string, id string) (*mq.DescribeUserOutput, error) {
	if brokerID == "" {
		return nil, &retry.NotFoundError{
			Message: "cannot find User with an empty broker ID.",
		}
	}
	if id == "" {
		return nil, &retry.NotFoundError{
			Message: "cannot find User with an empty username.",
		}
	}

	input := &mq.DescribeUserInput{
		BrokerId: aws.String(brokerID),
		Username: aws.String(id),
	}

	output, err := conn.DescribeUser(ctx, input)
	if errs.IsA[*awstypes.NotFoundException](err) {
		return nil, &retry.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}
	if err != nil {
		return nil, err
	}
	if output == nil {
		return nil, tfresource.NewEmptyResultError(input)
	}

	return output, nil
}

type resourceUserData struct {
	BrokerID        types.String                        `tfsdk:"broker_id"`
	ConsoleAccess   types.Bool                          `tfsdk:"console_access"`
	Groups          types.List                          `tfsdk:"groups"`
	ID              types.String                        `tfsdk:"id"`
	Password        types.String                        `tfsdk:"password"`
	ReplicationUser types.Bool                          `tfsdk:"replication_user"`
	Username        types.String                        `tfsdk:"username"`
	Pending         fwtypes.ObjectValueOf[pendingModel] `tfsdk:"pending"` // Computed-only field tracking pending AWS changes
}

// pendingModel represents pending changes to an MQ user. AWS MQ may return this
// information when modifications are queued but not yet applied.
// NOTE: The AWS API also returns a pending.groups field which is intentionally
// not included in this model. Future implementations may add support for it if
// needed.
type pendingModel struct {
	ConsoleAccess types.Bool                              `tfsdk:"console_access"`
	PendingChange fwtypes.StringEnum[awstypes.ChangeType] `tfsdk:"pending_change"`
}

func (rd *resourceUserData) refreshFromOutput(ctx context.Context, out *mq.DescribeUserOutput) diag.Diagnostics {
	if out == nil {
		return nil
	}

	rd.BrokerID = flex.StringToFramework(ctx, out.BrokerId)
	rd.ConsoleAccess = flex.BoolToFramework(ctx, out.ConsoleAccess)
	rd.Groups = flex.FlattenFrameworkStringValueList(ctx, out.Groups)
	rd.ReplicationUser = flex.BoolToFramework(ctx, out.ReplicationUser)
	rd.Username = flex.StringToFramework(ctx, out.Username)
	rd.ID = rd.Username

	// Populate the pending attribute with data from the AWS API response.
	// If there are pending changes, populate with actual values; otherwise use
	// an empty struct.
	if out.Pending != nil {
		pending, d := fwtypes.NewObjectValueOf(ctx, &pendingModel{
			ConsoleAccess: flex.BoolToFramework(ctx, out.Pending.ConsoleAccess),
			PendingChange: fwtypes.StringEnumValue(out.Pending.PendingChange),
		})
		if d.HasError() {
			return d
		}
		rd.Pending = pending
	} else {
		// Initialize with empty values when no pending changes exist.
		pending, d := fwtypes.NewObjectValueOf(ctx, &pendingModel{})
		if d.HasError() {
			return d
		}
		rd.Pending = pending
	}
	return nil
}

func userHasChanges(plan, state resourceUserData) bool {
	return !plan.ConsoleAccess.Equal(state.ConsoleAccess) ||
		!plan.Groups.Equal(state.Groups) ||
		!plan.Password.Equal(state.Password) ||
		!plan.ReplicationUser.Equal(state.ReplicationUser)
}

// Custom Plan Modifier: Sorts list items
type listSortModifier struct{}

func (m listSortModifier) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	// Get plan value
	planValue := req.PlanValue

	// If plan value is null or unknown, do nothing
	if planValue.IsNull() || planValue.IsUnknown() {
		return
	}

	// Convert plan value to []string
	var groups []string
	diags := planValue.ElementsAs(ctx, &groups, false)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	sort.Strings(groups)

	// Write sorted value back to plan
	sortedList, diags := types.ListValueFrom(ctx, types.StringType, groups)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.PlanValue = sortedList
}

func (m listSortModifier) Description(ctx context.Context) string {
	return "Sorts the list elements alphabetically."
}

func (m listSortModifier) MarkdownDescription(ctx context.Context) string {
	return "Sorts the list elements alphabetically."
}

// consoleAccessPlanModifier prevents unnecessary updates when the planned
// console_access value matches a pending change in AWS. This avoids triggering
// an update when the change is already queued on the AWS side, keeping the
// current state value instead.
type consoleAccessPlanModifier struct{}

func (c consoleAccessPlanModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	var state *resourceUserData
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state == nil || state.Pending.IsNull() {
		return
	}

	var pm pendingModel
	resp.Diagnostics.Append(state.Pending.As(ctx, &pm, basetypes.ObjectAsOptions{})...)

	consoleAccessValue := req.PlanValue
	// If the planned value matches the pending console_access, keep the current
	// state value to avoid an unnecessary update operation.
	if !pm.ConsoleAccess.IsNull() && !pm.ConsoleAccess.IsUnknown() && pm.ConsoleAccess.Equal(consoleAccessValue) {
		resp.PlanValue = req.StateValue
	}
}

func (c consoleAccessPlanModifier) Description(_ context.Context) string {
	return "Prevents unnecessary updates when the planned console_access value matches a pending change in AWS."
}

func (c consoleAccessPlanModifier) MarkdownDescription(_ context.Context) string {
	return "Prevents unnecessary updates when the planned `console_access` value matches a pending change in AWS."
}

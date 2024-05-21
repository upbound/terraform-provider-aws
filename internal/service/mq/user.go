// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package mq

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/mq"
	awstypes "github.com/aws/aws-sdk-go-v2/service/mq/types"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	"github.com/hashicorp/terraform-provider-aws/internal/framework/flex"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

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
			},
			"groups": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
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
		response.Diagnostics.Append(create.DiagErrorFramework(names.MQ, create.ErrActionCreating, ResourceNameUser, fmt.Sprintf("%s/%s", plan.BrokerID.ValueString(), plan.Username.ValueString()), err))
		return
	}

	// Create API call returns no data. Get resource details.
	userDetails, err := findUserByID(ctx, conn, plan.BrokerID.ValueString(), plan.Username.ValueString())
	if err != nil {
		response.Diagnostics.Append(create.DiagErrorFramework(names.MQ, create.ErrActionCreating, ResourceNameUser, fmt.Sprintf("%s/%s", plan.BrokerID.ValueString(), plan.Username.ValueString()), err))
		return
	}

	state := plan
	state.refreshFromOutput(ctx, userDetails)

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
		response.Diagnostics.Append(create.DiagErrorFramework(names.MQ, create.ErrActionReading, ResourceNameUser, fmt.Sprintf("%s/%s", state.BrokerID.ValueString(), state.ID.ValueString()), err))
		return
	}

	state.refreshFromOutput(ctx, userDetails)
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
			response.Diagnostics.Append(create.DiagErrorFramework(names.MQ, create.ErrActionUpdating, ResourceNameUser, fmt.Sprintf("%s/%s", state.BrokerID.ValueString(), state.ID.ValueString()), err))
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
		response.Diagnostics.Append(create.DiagErrorFramework(names.MQ, create.ErrActionDeleting, ResourceNameUser, fmt.Sprintf("%s/%s", state.BrokerID.ValueString(), state.ID.ValueString()), err))
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
	BrokerID      types.String `tfsdk:"broker_id"`
	ConsoleAccess types.Bool   `tfsdk:"console_access"`
	Groups        types.List   `tfsdk:"groups"`
	ID            types.String `tfsdk:"id"`
	Password      types.String `tfsdk:"password"`
	// Pending         types.Object `tfsdk:"pending"`
	ReplicationUser types.Bool   `tfsdk:"replication_user"`
	Username        types.String `tfsdk:"username"`
}

func (rd *resourceUserData) refreshFromOutput(ctx context.Context, out *mq.DescribeUserOutput) {
	if out == nil {
		return
	}

	rd.BrokerID = flex.StringToFramework(ctx, out.BrokerId)
	rd.ConsoleAccess = flex.BoolToFramework(ctx, out.ConsoleAccess)
	rd.Groups = flex.FlattenFrameworkStringValueList(ctx, out.Groups)
	rd.ReplicationUser = flex.BoolToFramework(ctx, out.ReplicationUser)
	rd.Username = flex.StringToFramework(ctx, out.Username)
	rd.ID = rd.Username
}

func userHasChanges(plan, state resourceUserData) bool {
	return !plan.ConsoleAccess.Equal(state.ConsoleAccess) ||
		!plan.Groups.Equal(state.Groups) ||
		!plan.Password.Equal(state.Password) ||
		!plan.ReplicationUser.Equal(state.ReplicationUser)
}

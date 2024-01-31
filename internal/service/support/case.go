// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package support

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/support"
	awstypes "github.com/aws/aws-sdk-go-v2/service/support/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
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

// @FrameworkResource(name="Case")
func newResourceCase(_ context.Context) (resource.ResourceWithConfigure, error) {
	return &resourceCase{}, nil
}

const (
	ResourceNameCase = "Case"
)

type resourceCase struct {
	framework.ResourceWithConfigure
}

func (r *resourceCase) Metadata(_ context.Context, _ resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = "aws_support_case"
}

// Schema returns the schema for this resource.
func (r *resourceCase) Schema(ctx context.Context, request resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"case_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"category_code": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cc_email_addresses": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"communication_body": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": framework.IDAttribute(),
			"issue_type": schema.StringAttribute{
				Optional: true,
			},
			"language": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service_code": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"severity_code": schema.StringAttribute{
				Required: true,
			},
			"subject": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *resourceCase) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var plan resourceCaseData
	response.Diagnostics.Append(request.Plan.Get(ctx, &plan)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SupportClient(ctx)

	input := support.CreateCaseInput{
		CommunicationBody: flex.StringFromFramework(ctx, plan.CommunicationBody),
		Subject:           flex.StringFromFramework(ctx, plan.Subject),
		CategoryCode:      flex.StringFromFramework(ctx, plan.CategoryCode),
		CcEmailAddresses:  flex.ExpandFrameworkStringValueList(ctx, plan.CCEmailAddresses),
		IssueType:         flex.StringFromFramework(ctx, plan.IssueType),
		Language:          flex.StringFromFramework(ctx, plan.Language),
		ServiceCode:       flex.StringFromFramework(ctx, plan.ServiceCode),
		SeverityCode:      flex.StringFromFramework(ctx, plan.SeverityCode),
	}
	output, err := conn.CreateCase(ctx, &input)
	if err != nil {
		response.Diagnostics.Append(create.DiagErrorFramework(names.Support, create.ErrActionCreating, ResourceNameCase, plan.Subject.String(), err))
		return
	}

	// Create API call returns only Case ID. Get other details as well.
	caseDetails, err := findCaseByID(ctx, conn, *output.CaseId)
	if err != nil {
		response.Diagnostics.Append(create.DiagErrorFramework(names.Support, create.ErrActionChecking, ResourceNameCase, plan.Subject.String(), err))
		return
	}

	state := plan
	state.refreshFromOutput(ctx, caseDetails)

	response.Diagnostics.Append(response.State.Set(ctx, state)...)
}

func (r *resourceCase) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var state resourceCaseData
	response.Diagnostics.Append(request.State.Get(ctx, &state)...)
	if response.Diagnostics.HasError() {
		return
	}

	conn := r.Meta().SupportClient(ctx)

	caseDetails, err := findCaseByID(ctx, conn, state.ID.ValueString())
	if tfresource.NotFound(err) {
		create.LogNotFoundRemoveState(names.Support, create.ErrActionReading, ResourceNameCase, state.ID.ValueString())
		response.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		response.Diagnostics.Append(create.DiagErrorFramework(names.Support, create.ErrActionReading, ResourceNameCase, state.ID.String(), err))
		return
	}

	state.refreshFromOutput(ctx, caseDetails)
	response.Diagnostics.Append(response.State.Set(ctx, &state)...)
}

// Update is a no-op.
func (r *resourceCase) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
}

// Delete is a no-op, because AWS doesn't provide a deletion API.
func (r *resourceCase) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *resourceCase) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
}

func findCaseByID(ctx context.Context, conn *support.Client, caseID string) (*awstypes.CaseDetails, error) {
	if caseID == "" {
		return nil, &retry.NotFoundError{
			Message: "cannot find Case with an empty ID.",
		}
	}

	input := &support.DescribeCasesInput{
		CaseIdList: []string{*aws.String(caseID)},
	}

	output, err := conn.DescribeCases(ctx, input)

	if errs.IsA[*awstypes.CaseIdNotFound](err) {
		return nil, &retry.NotFoundError{
			LastError:   err,
			LastRequest: input,
		}
	}

	if err != nil {
		return nil, err
	}

	if output == nil || len(output.Cases) == 0 {
		return nil, tfresource.NewEmptyResultError(input)
	}

	if len(output.Cases) > 1 {
		return nil, tfresource.NewTooManyResultsError(len(output.Cases), input)
	}

	return &output.Cases[0], nil
}

type resourceCaseData struct {
	CaseID            types.String `tfsdk:"case_id"`
	CategoryCode      types.String `tfsdk:"category_code"`
	CCEmailAddresses  types.List   `tfsdk:"cc_email_addresses"`
	CommunicationBody types.String `tfsdk:"communication_body"`
	DisplayID         types.String `tfsdk:"display_id"`
	ID                types.String `tfsdk:"id"`
	IssueType         types.String `tfsdk:"issue_type"`
	Language          types.String `tfsdk:"language"`
	ServiceCode       types.String `tfsdk:"service_code"`
	SeverityCode      types.String `tfsdk:"severity_code"`
	Subject           types.String `tfsdk:"subject"`
}

func (rd *resourceCaseData) refreshFromOutput(ctx context.Context, out *awstypes.CaseDetails) {
	if out == nil {
		return
	}

	rd.CaseID = flex.StringToFramework(ctx, out.CaseId)
	rd.ID = rd.CaseID

	rd.CategoryCode = flex.StringToFramework(ctx, out.CategoryCode)
	rd.CCEmailAddresses = flex.FlattenFrameworkStringValueList(ctx, out.CcEmailAddresses)
	rd.DisplayID = flex.StringToFramework(ctx, out.DisplayId)
	rd.Language = flex.StringToFramework(ctx, out.Language)
	rd.ServiceCode = flex.StringToFramework(ctx, out.ServiceCode)
	rd.SeverityCode = flex.StringToFramework(ctx, out.SeverityCode)
	rd.Subject = flex.StringToFramework(ctx, out.Subject)
}

package provider

import (
	"context"

	"github.com/gringolito/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type trueNASService struct {
	ctx    context.Context
	client *client.Client
	diags  diag.Diagnostics
}

func (s *trueNASService) handleAPIError(err error) {
	if e, ok := err.(client.APIError); ok {
		s.diags.AddError(e.Msg, e.Err.Error())
	} else {
		s.diags.AddError("TrueNAS API Error", err.Error())
	}
}

func (s *trueNASService) stringArrayFromList(list basetypes.ListValue) []string {
	elements := make([]types.String, 0, len(list.Elements()))
	s.diags.Append(list.ElementsAs(s.ctx, &elements, false)...)
	array := make([]string, 0, len(elements))
	for _, element := range elements {
		array = append(array, element.ValueString())
	}
	return array
}

func (s *trueNASService) int64ArrayFromList(list basetypes.ListValue) []int64 {
	elements := make([]types.Int64, 0, len(list.Elements()))
	s.diags.Append(list.ElementsAs(s.ctx, &elements, false)...)
	array := make([]int64, 0, len(elements))
	for _, element := range elements {
		array = append(array, element.ValueInt64())
	}
	return array
}

func (s *trueNASService) listValueFrom(elementType attr.Type, elements any) basetypes.ListValue {
	list, diags := types.ListValueFrom(s.ctx, elementType, elements)
	s.diags.Append(diags...)
	return list
}

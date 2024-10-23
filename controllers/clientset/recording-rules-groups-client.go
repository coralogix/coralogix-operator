package clientset

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

//go:generate mockgen -destination=../mock_clientset/mock_recordingrulesgroups-client.go -package=mock_clientset -source=recording-rules-groups-client.go RecordingRulesGroupsClientInterface
type RecordingRulesGroupsClientInterface interface {
	Create(ctx context.Context, req *cxsdk.CreateRuleGroupSetRequest) (*cxsdk.CreateRuleGroupSetResponse, error)
	Get(ctx context.Context, req *cxsdk.GetRuleGroupSetRequest) (*cxsdk.GetRuleGroupSetResponse, error)
	Update(ctx context.Context, req *cxsdk.UpdateRuleGroupSetRequest) (*emptypb.Empty, error)
	Delete(ctx context.Context, req *cxsdk.DeleteRuleGroupSetRequest) (*emptypb.Empty, error)
}

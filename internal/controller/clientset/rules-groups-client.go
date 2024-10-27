package clientset

import (
	"context"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
)

//go:generate mockgen -destination=../mock_clientset/mock_rulegroups-client.go -package=mock_clientset -source=rules-groups-client.go RuleGroupsClientInterface
type RuleGroupsClientInterface interface {
	Create(ctx context.Context, req *cxsdk.CreateRuleGroupRequest) (*cxsdk.CreateRuleGroupResponse, error)
	Get(ctx context.Context, req *cxsdk.GetRuleGroupRequest) (*cxsdk.GetRuleGroupResponse, error)
	Update(ctx context.Context, req *cxsdk.UpdateRuleGroupRequest) (*cxsdk.UpdateRuleGroupResponse, error)
	Delete(ctx context.Context, req *cxsdk.DeleteRuleGroupRequest) (*cxsdk.DeleteRuleGroupResponse, error)
}

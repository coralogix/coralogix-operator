package clientset

import (
	"context"

	rulesgroups "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/rules-groups/v1"
)

//go:generate mockgen -destination=../mock_clientset/mock_rulegroups-client.go -package=mock_clientset github.com/coralogix/coralogix-operator/controllers/clientset RuleGroupsClientInterface
type RuleGroupsClientInterface interface {
	CreateRuleGroup(ctx context.Context, req *rulesgroups.CreateRuleGroupRequest) (*rulesgroups.CreateRuleGroupResponse, error)
	GetRuleGroup(ctx context.Context, req *rulesgroups.GetRuleGroupRequest) (*rulesgroups.GetRuleGroupResponse, error)
	UpdateRuleGroup(ctx context.Context, req *rulesgroups.UpdateRuleGroupRequest) (*rulesgroups.UpdateRuleGroupResponse, error)
	DeleteRuleGroup(ctx context.Context, req *rulesgroups.DeleteRuleGroupRequest) (*rulesgroups.DeleteRuleGroupResponse, error)
}

type RuleGroupsClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

func (r RuleGroupsClient) CreateRuleGroup(ctx context.Context, req *rulesgroups.CreateRuleGroupRequest) (*rulesgroups.CreateRuleGroupResponse, error) {
	callProperties, err := r.callPropertiesCreator.GetCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := rulesgroups.NewRuleGroupsServiceClient(conn)

	return client.CreateRuleGroup(callProperties.Ctx, req, callProperties.CallOptions...)
}

func (r RuleGroupsClient) GetRuleGroup(ctx context.Context, req *rulesgroups.GetRuleGroupRequest) (*rulesgroups.GetRuleGroupResponse, error) {
	callProperties, err := r.callPropertiesCreator.GetCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := rulesgroups.NewRuleGroupsServiceClient(conn)

	return client.GetRuleGroup(callProperties.Ctx, req, callProperties.CallOptions...)
}

func (r RuleGroupsClient) UpdateRuleGroup(ctx context.Context, req *rulesgroups.UpdateRuleGroupRequest) (*rulesgroups.UpdateRuleGroupResponse, error) {
	callProperties, err := r.callPropertiesCreator.GetCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()

	client := rulesgroups.NewRuleGroupsServiceClient(conn)

	return client.UpdateRuleGroup(callProperties.Ctx, req, callProperties.CallOptions...)
}

func (r RuleGroupsClient) DeleteRuleGroup(ctx context.Context, req *rulesgroups.DeleteRuleGroupRequest) (*rulesgroups.DeleteRuleGroupResponse, error) {
	callProperties, err := r.callPropertiesCreator.GetCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()

	client := rulesgroups.NewRuleGroupsServiceClient(conn)

	return client.DeleteRuleGroup(callProperties.Ctx, req, callProperties.CallOptions...)
}

func NewRuleGroupsClient(c *CallPropertiesCreator) *RuleGroupsClient {
	return &RuleGroupsClient{callPropertiesCreator: c}
}

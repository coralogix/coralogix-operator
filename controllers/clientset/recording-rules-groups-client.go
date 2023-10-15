package clientset

import (
	"context"

	rrg "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/recording-rules-groups/v2"
	"google.golang.org/protobuf/types/known/emptypb"
)

//go:generate mockgen -destination=../mock_clientset/mock_recordingrulesgroups-client.go -package=mock_clientset github.com/coralogix/coralogix-operator/controllers/clientset RecordingRulesGroupsClientInterface
type RecordingRulesGroupsClientInterface interface {
	CreateRecordingRuleGroupSet(ctx context.Context, req *rrg.CreateRuleGroupSet) (*rrg.CreateRuleGroupSetResult, error)
	GetRecordingRuleGroupSet(ctx context.Context, req *rrg.FetchRuleGroupSet) (*rrg.OutRuleGroupSet, error)
	UpdateRecordingRuleGroupSet(ctx context.Context, req *rrg.UpdateRuleGroupSet) (*emptypb.Empty, error)
	DeleteRecordingRuleGroupSet(ctx context.Context, req *rrg.DeleteRuleGroupSet) (*emptypb.Empty, error)
}

type RecordingRulesGroupsClient struct {
	callPropertiesCreator *CallPropertiesCreator
}

func (r RecordingRulesGroupsClient) CreateRecordingRuleGroupSet(ctx context.Context, req *rrg.CreateRuleGroupSet) (*rrg.CreateRuleGroupSetResult, error) {
	callProperties, err := r.callPropertiesCreator.GetCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := rrg.NewRuleGroupSetsClient(conn)

	ctx = createAuthContext(ctx, r.callPropertiesCreator.apiKey)
	return client.Create(callProperties.Ctx, req, callProperties.CallOptions...)
}

func (r RecordingRulesGroupsClient) GetRecordingRuleGroupSet(ctx context.Context, req *rrg.FetchRuleGroupSet) (*rrg.OutRuleGroupSet, error) {
	callProperties, err := r.callPropertiesCreator.GetCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := rrg.NewRuleGroupSetsClient(conn)

	return client.Fetch(callProperties.Ctx, req, callProperties.CallOptions...)
}

func (r RecordingRulesGroupsClient) UpdateRecordingRuleGroupSet(ctx context.Context, req *rrg.UpdateRuleGroupSet) (*emptypb.Empty, error) {
	callProperties, err := r.callPropertiesCreator.GetCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := rrg.NewRuleGroupSetsClient(conn)

	return client.Update(callProperties.Ctx, req, callProperties.CallOptions...)
}

func (r RecordingRulesGroupsClient) DeleteRecordingRuleGroupSet(ctx context.Context, req *rrg.DeleteRuleGroupSet) (*emptypb.Empty, error) {
	callProperties, err := r.callPropertiesCreator.GetCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := rrg.NewRuleGroupSetsClient(conn)

	return client.Delete(callProperties.Ctx, req, callProperties.CallOptions...)
}

func (r RecordingRulesGroupsClient) ListRecordingRuleGroup(ctx context.Context) (*rrg.RuleGroupSetListing, error) {
	callProperties, err := r.callPropertiesCreator.GetCallProperties(ctx)
	if err != nil {
		return nil, err
	}

	conn := callProperties.Connection
	defer conn.Close()
	client := rrg.NewRuleGroupSetsClient(conn)

	ctx = createAuthContext(ctx, r.callPropertiesCreator.apiKey)
	return client.List(callProperties.Ctx, &emptypb.Empty{}, callProperties.CallOptions...)
}

func NewRecordingRuleGroupsClient(c *CallPropertiesCreator) *RecordingRulesGroupsClient {
	return &RecordingRulesGroupsClient{callPropertiesCreator: c}
}

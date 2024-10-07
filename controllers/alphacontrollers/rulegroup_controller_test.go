package alphacontrollers

import (
	"context"
	"fmt"
	"testing"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/controllers/clientset"
	"github.com/coralogix/coralogix-operator/controllers/mock_clientset"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var ruleGroupBackendSchema = &cxsdk.RuleGroup{
	Id:           wrapperspb.String("id"),
	Name:         wrapperspb.String("name"),
	Description:  wrapperspb.String("description"),
	Creator:      wrapperspb.String("creator"),
	Enabled:      wrapperspb.Bool(true),
	Hidden:       wrapperspb.Bool(false),
	RuleMatchers: []*cxsdk.RuleMatcher{},
	RuleSubgroups: []*cxsdk.RuleSubgroup{
		{
			Id:    wrapperspb.String("subgroup_id"),
			Order: wrapperspb.UInt32(1),
			Rules: []*cxsdk.Rule{
				{
					Id:          wrapperspb.String("rule_id"),
					Name:        wrapperspb.String("rule_name"),
					Description: wrapperspb.String("rule_description"),
					SourceField: wrapperspb.String("text"),
					Parameters: &cxsdk.RuleParameters{
						RuleParameters: &cxsdk.RuleParametersJSONExtractParameters{
							JsonExtractParameters: &cxsdk.JSONExtractParameters{
								DestinationField: cxsdk.JSONExtractParametersDestinationFieldSeverity,
								Rule:             wrapperspb.String(`{"severity": "info"}`),
							},
						},
					},
					Enabled: wrapperspb.Bool(true),
					Order:   wrapperspb.UInt32(3),
				},
			},
		},
	},
	Order: wrapperspb.UInt32(1),
}

func expectedRuleGroupCRD() *coralogixv1alpha1.RuleGroup {
	return &coralogixv1alpha1.RuleGroup{
		TypeMeta:   metav1.TypeMeta{Kind: "RecordingRuleGroupSet", APIVersion: "coralogix.com/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: coralogixv1alpha1.RuleGroupSpec{
			Name:        "name",
			Description: "description",
			Creator:     "creator",
			Active:      true,
			Hidden:      false,
			Order:       pointer.Int32(1),
			RuleSubgroups: []coralogixv1alpha1.RuleSubGroup{
				{
					Order: pointer.Int32(1),
					Rules: []coralogixv1alpha1.Rule{
						{
							Name:        "rule_name",
							Description: "rule_description",
							Active:      true,
							JsonExtract: &coralogixv1alpha1.JsonExtract{
								DestinationField: coralogixv1alpha1.DestinationFieldRuleSeverity,
								JsonKey:          "{\"severity\": \"info\"}",
							},
						},
					},
				},
			},
		},
	}
}

func TestFlattenRuleGroupsErrorsOnBadResponse(t *testing.T) {
	ruleGroup := &cxsdk.RuleGroup{
		Id:           wrapperspb.String("id"),
		Name:         wrapperspb.String("name"),
		Description:  wrapperspb.String("description"),
		Creator:      wrapperspb.String("creator"),
		Enabled:      wrapperspb.Bool(true),
		Hidden:       wrapperspb.Bool(false),
		RuleMatchers: []*cxsdk.RuleMatcher{},
		RuleSubgroups: []*cxsdk.RuleSubgroup{
			{
				Rules: []*cxsdk.Rule{
					{
						Id:          wrapperspb.String("rule_id"),
						Name:        wrapperspb.String("rule_name"),
						Description: wrapperspb.String("rule_description"),
						SourceField: wrapperspb.String("text"),
						Parameters: &cxsdk.RuleParameters{
							RuleParameters: nil,
						},
						Enabled: wrapperspb.Bool(true),
						Order:   wrapperspb.UInt32(1),
					},
				},
			},
		},
		Order: wrapperspb.UInt32(1),
	}

	status, err := flattenRuleGroup(ruleGroup)
	assert.Error(t, err)
	assert.Nil(t, status)
}

func TestFlattenRuleGroups(t *testing.T) {
	actualStatus, err := flattenRuleGroup(ruleGroupBackendSchema)
	assert.NoError(t, err)

	id := "id"
	subgroupId := "subgroup_id"
	expectedStatus := &coralogixv1alpha1.RuleGroupStatus{
		ID:           &id,
		Name:         "name",
		Description:  "description",
		Active:       true,
		Applications: nil,
		Subsystems:   nil,
		Severities:   nil,
		Hidden:       false,
		Creator:      "creator",
		Order:        pointer.Int32(1),
		RuleSubgroups: []coralogixv1alpha1.RuleSubGroup{
			{
				ID:     &subgroupId,
				Active: false,
				Order:  pointer.Int32(1),
				Rules: []coralogixv1alpha1.Rule{
					{
						Name:        "rule_name",
						Description: "rule_description",
						Active:      true,
						Parse:       nil,
						Block:       nil,
						JsonExtract: &coralogixv1alpha1.JsonExtract{
							DestinationField: coralogixv1alpha1.DestinationFieldRuleSeverity,
							JsonKey:          "{\"severity\": \"info\"}",
						},
					},
				},
			},
		},
	}

	assert.Equal(t, expectedStatus, actualStatus)
}

func TestRuleGroupReconciler_Reconcile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	ruleGroupClient := createRuleGroupClientSimpleMock(mockCtrl)
	mockClientSet := mock_clientset.NewMockClientSetInterface(mockCtrl)
	mockClientSet.EXPECT().RuleGroups().Return(ruleGroupClient).AnyTimes()

	scheme := runtime.NewScheme()
	utilruntime.Must(coralogixv1alpha1.AddToScheme(scheme))
	mgr, _ := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go mgr.GetCache().Start(ctx)
	mgr.GetCache().WaitForCacheSync(ctx)
	withWatch, err := client.NewWithWatch(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
	})
	assert.NoError(t, err)
	r := RuleGroupReconciler{
		Client:             withWatch,
		Scheme:             mgr.GetScheme(),
		CoralogixClientSet: mockClientSet,
	}
	r.SetupWithManager(mgr)

	watcher, _ := r.Client.(client.WithWatch).Watch(ctx, &coralogixv1alpha1.RuleGroupList{})
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	err = r.Client.Create(ctx, expectedRuleGroupCRD())
	assert.NoError(t, err)
	<-watcher.ResultChan()

	result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
	assert.NoError(t, err)

	namespacedName := types.NamespacedName{Namespace: "default", Name: "test"}
	actualRuleGroupCRD := &coralogixv1alpha1.RuleGroup{}
	err = r.Client.Get(ctx, namespacedName, actualRuleGroupCRD)
	assert.NoError(t, err)

	id := actualRuleGroupCRD.Status.ID
	if !assert.NotNil(t, id) {
		return
	}
	getRuleGroupRequest := &cxsdk.GetRuleGroupRequest{GroupId: *id}
	actualRuleGroup, err := r.CoralogixClientSet.RuleGroups().Get(ctx, getRuleGroupRequest)
	assert.NoError(t, err)
	assert.EqualValues(t, ruleGroupBackendSchema, actualRuleGroup.GetRuleGroup())

	err = r.Client.Delete(ctx, actualRuleGroupCRD)
	<-watcher.ResultChan()

	result, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
	assert.NoError(t, err)
	assert.Equal(t, false, result.Requeue)

	actualRuleGroup, err = r.CoralogixClientSet.RuleGroups().Get(ctx, getRuleGroupRequest)
	assert.Nil(t, actualRuleGroup)
	assert.Error(t, err)
}

func TestRuleGroupReconciler_Reconcile_5XX_StatusError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	ruleGroupClient := createRecordingRuleGroupClientSimpleMockWith5XXStatusError(mockCtrl)
	mockClientSet := mock_clientset.NewMockClientSetInterface(mockCtrl)
	mockClientSet.EXPECT().RuleGroups().Return(ruleGroupClient).AnyTimes()

	scheme := runtime.NewScheme()
	utilruntime.Must(coralogixv1alpha1.AddToScheme(scheme))
	mgr, _ := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go mgr.GetCache().Start(ctx)
	mgr.GetCache().WaitForCacheSync(ctx)
	withWatch, err := client.NewWithWatch(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
	})
	assert.NoError(t, err)
	r := RuleGroupReconciler{
		Client:             withWatch,
		Scheme:             mgr.GetScheme(),
		CoralogixClientSet: mockClientSet,
	}
	r.SetupWithManager(mgr)

	watcher, _ := r.Client.(client.WithWatch).Watch(ctx, &coralogixv1alpha1.RuleGroupList{})
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	err = r.Client.Create(ctx, expectedRuleGroupCRD())
	assert.NoError(t, err)
	<-watcher.ResultChan()

	result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
	assert.Error(t, err)
	assert.Equal(t, defaultErrRequeuePeriod, result.RequeueAfter)

	result, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
	assert.NoError(t, err)

	namespacedName := types.NamespacedName{Namespace: "default", Name: "test"}
	actualRuleGroupCRD := &coralogixv1alpha1.RuleGroup{}
	err = r.Client.Get(ctx, namespacedName, actualRuleGroupCRD)
	assert.NoError(t, err)

	id := actualRuleGroupCRD.Status.ID
	if !assert.NotNil(t, id) {
		return
	}
	getRuleGroupRequest := &cxsdk.GetRuleGroupRequest{GroupId: *id}
	actualRuleGroup, err := r.CoralogixClientSet.RuleGroups().Get(ctx, getRuleGroupRequest)
	assert.NoError(t, err)
	assert.EqualValues(t, ruleGroupBackendSchema, actualRuleGroup.GetRuleGroup())

	err = r.Client.Delete(ctx, actualRuleGroupCRD)
	<-watcher.ResultChan()
	r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
}

// createRuleGroupClientSimpleMock creates a simple mock for RuleGroupsClientInterface which returns a single rule group.
func createRuleGroupClientSimpleMock(mockCtrl *gomock.Controller) clientset.RuleGroupsClientInterface {
	mockRuleGroupsClient := mock_clientset.NewMockRuleGroupsClientInterface(mockCtrl)

	var ruleGroupExist bool

	mockRuleGroupsClient.EXPECT().
		Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *cxsdk.CreateRuleGroupRequest) (*cxsdk.CreateRuleGroupResponse, error) {
		ruleGroupExist = true
		return &cxsdk.CreateRuleGroupResponse{RuleGroup: ruleGroupBackendSchema}, nil
	}).AnyTimes()

	mockRuleGroupsClient.EXPECT().
		Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, req *cxsdk.GetRuleGroupRequest) (*cxsdk.GetRuleGroupResponse, error) {
		if ruleGroupExist {
			return &cxsdk.GetRuleGroupResponse{RuleGroup: ruleGroupBackendSchema}, nil
		}
		return nil, errors.NewNotFound(schema.GroupResource{}, "id1")
	}).AnyTimes()

	mockRuleGroupsClient.EXPECT().
		Delete(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *cxsdk.DeleteRuleGroupRequest) (*cxsdk.DeleteRuleGroupResponse, error) {
		if ruleGroupExist {
			ruleGroupExist = false
			return &cxsdk.DeleteRuleGroupResponse{}, nil
		}
		return nil, errors.NewNotFound(schema.GroupResource{}, "id1")
	}).AnyTimes()

	return mockRuleGroupsClient
}

// createRecordingRuleGroupClientSimpleMockWith5XXStatusError creates a simple mock for RecordingRuleGroupsClientInterface which first returns 5xx status error, then returns a single recording rule group.
func createRecordingRuleGroupClientSimpleMockWith5XXStatusError(mockCtrl *gomock.Controller) clientset.RuleGroupsClientInterface {
	mockRuleGroupsClient := mock_clientset.NewMockRuleGroupsClientInterface(mockCtrl)

	var ruleGroupExist, wasCalled bool

	mockRuleGroupsClient.EXPECT().
		Create(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *cxsdk.CreateRuleGroupRequest) (*cxsdk.CreateRuleGroupResponse, error) {
		if !wasCalled {
			wasCalled = true
			return nil, errors.NewInternalError(fmt.Errorf("internal error"))
		}
		ruleGroupExist = true
		return &cxsdk.CreateRuleGroupResponse{RuleGroup: ruleGroupBackendSchema}, nil
	}).AnyTimes()

	mockRuleGroupsClient.EXPECT().
		Get(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, req *cxsdk.GetRuleGroupRequest) (*cxsdk.GetRuleGroupResponse, error) {
		if ruleGroupExist {
			return &cxsdk.GetRuleGroupResponse{RuleGroup: ruleGroupBackendSchema}, nil
		}
		return nil, errors.NewNotFound(schema.GroupResource{}, "id1")
	}).AnyTimes()

	mockRuleGroupsClient.EXPECT().
		Delete(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *cxsdk.DeleteRuleGroupRequest) (*cxsdk.DeleteRuleGroupResponse, error) {
		if ruleGroupExist {
			ruleGroupExist = false
			return &cxsdk.DeleteRuleGroupResponse{}, nil
		}
		return nil, errors.NewNotFound(schema.GroupResource{}, "id1")
	}).AnyTimes()

	return mockRuleGroupsClient
}

package alphacontrollers

import (
	"context"
	"fmt"
	"testing"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/controllers/clientset"
	rrg "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/recording-rules-groups/v2"
	"github.com/coralogix/coralogix-operator/controllers/mock_clientset"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/emptypb"
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

var recordingRuleGroupSetBackendSchema = &rrg.OutRuleGroupSet{
	Id: "id1",
	Groups: []*rrg.OutRuleGroup{
		{
			Name:     "name",
			Interval: pointer.Uint32(60),
			Limit:    pointer.Uint64(100),
			Rules: []*rrg.OutRule{
				{
					Record: "record",
					Expr:   "vector(1)",
					Labels: map[string]string{"key": "value"},
				},
			},
		},
	},
}

func expectedRecordingRuleGroupSetCRD() *coralogixv1alpha1.RecordingRuleGroupSet {
	return &coralogixv1alpha1.RecordingRuleGroupSet{
		TypeMeta:   metav1.TypeMeta{Kind: "RecordingRuleGroupSet", APIVersion: "coralogix.com/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: coralogixv1alpha1.RecordingRuleGroupSetSpec{
			Groups: []coralogixv1alpha1.RecordingRuleGroup{
				{
					Name:            "name",
					IntervalSeconds: 60,
					Limit:           100,
					Rules: []coralogixv1alpha1.RecordingRule{
						{
							Record: "record",
							Expr:   "vector(1)",
							Labels: map[string]string{"key": "value"},
						},
					},
				},
			},
		},
	}
}

func TestFlattenRecordingRuleGroupSet(t *testing.T) {
	actualStatus := flattenRecordingRuleGroupSet(recordingRuleGroupSetBackendSchema)
	expectedStatus := coralogixv1alpha1.RecordingRuleGroupSetStatus{
		ID: pointer.String("id1"),
		Groups: []coralogixv1alpha1.RecordingRuleGroup{
			{
				Name:            "name",
				IntervalSeconds: 60,
				Limit:           100,
				Rules: []coralogixv1alpha1.RecordingRule{
					{
						Record: "record",
						Expr:   "vector(1)",
						Labels: map[string]string{"key": "value"},
					},
				},
			},
		},
	}
	assert.EqualValues(t, expectedStatus, actualStatus)
}

func TestRecordingRuleGroupSetReconciler_Reconcile(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	recordingRuleGroupClient := createRecordingRuleGroupClientSimpleMock(mockCtrl)
	mockClientSet := mock_clientset.NewMockClientSetInterface(mockCtrl)
	mockClientSet.EXPECT().RecordingRuleGroups().Return(recordingRuleGroupClient).AnyTimes()

	scheme := runtime.NewScheme()
	utilruntime.Must(coralogixv1alpha1.AddToScheme(scheme))
	mgr, _ := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: ":8081",
	})
	ctx := context.Background()
	go mgr.GetCache().Start(ctx)
	mgr.GetCache().WaitForCacheSync(ctx)
	withWatch, err := client.NewWithWatch(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
	})
	assert.NoError(t, err)
	r := RecordingRuleGroupSetReconciler{
		Client:             withWatch,
		Scheme:             mgr.GetScheme(),
		CoralogixClientSet: mockClientSet,
	}
	r.SetupWithManager(mgr)

	watcher, _ := r.Client.(client.WithWatch).Watch(ctx, &coralogixv1alpha1.RecordingRuleGroupSetList{})
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	err = r.Client.Create(ctx, expectedRecordingRuleGroupSetCRD())
	assert.NoError(t, err)
	<-watcher.ResultChan()

	result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
	assert.NoError(t, err)
	assert.Equal(t, defaultRequeuePeriod, result.RequeueAfter)

	namespacedName := types.NamespacedName{Namespace: "default", Name: "test"}
	actualRecordingRuleGroupSetCRD := &coralogixv1alpha1.RecordingRuleGroupSet{}
	err = r.Client.Get(ctx, namespacedName, actualRecordingRuleGroupSetCRD)
	assert.NoError(t, err)

	id := actualRecordingRuleGroupSetCRD.Status.ID
	if !assert.NotNil(t, id) {
		return
	}
	getRecordingRuleGroupSetRequest := &rrg.FetchRuleGroupSet{Id: *id}
	actualRecordingRuleGroupSet, err := r.CoralogixClientSet.RecordingRuleGroups().GetRecordingRuleGroupSet(ctx, getRecordingRuleGroupSetRequest)
	assert.NoError(t, err)
	assert.EqualValues(t, recordingRuleGroupSetBackendSchema, actualRecordingRuleGroupSet)

	err = r.Client.Delete(ctx, actualRecordingRuleGroupSetCRD)
	<-watcher.ResultChan()

	result, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
	assert.NoError(t, err)
	assert.Equal(t, false, result.Requeue)

	actualRecordingRuleGroupSet, err = r.CoralogixClientSet.RecordingRuleGroups().GetRecordingRuleGroupSet(ctx, getRecordingRuleGroupSetRequest)
	assert.Nil(t, actualRecordingRuleGroupSet)
	assert.Error(t, err)
}

func TestRecordingRuleGroupSetReconciler_Reconcile_5XX_StatusError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	recordingRuleGroupClient := createRecordingRuleGroupClientMockWith5XXStatusError(mockCtrl)
	mockClientSet := mock_clientset.NewMockClientSetInterface(mockCtrl)
	mockClientSet.EXPECT().RecordingRuleGroups().Return(recordingRuleGroupClient).AnyTimes()

	scheme := runtime.NewScheme()
	utilruntime.Must(coralogixv1alpha1.AddToScheme(scheme))
	mgr, _ := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: ":8081",
	})
	ctx := context.Background()
	go mgr.GetCache().Start(ctx)
	mgr.GetCache().WaitForCacheSync(ctx)
	withWatch, err := client.NewWithWatch(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
	})
	assert.NoError(t, err)
	r := RecordingRuleGroupSetReconciler{
		Client:             withWatch,
		Scheme:             mgr.GetScheme(),
		CoralogixClientSet: mockClientSet,
	}
	r.SetupWithManager(mgr)

	watcher, _ := r.Client.(client.WithWatch).Watch(ctx, &coralogixv1alpha1.RecordingRuleGroupSetList{})
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	err = r.Client.Create(ctx, expectedRecordingRuleGroupSetCRD())
	assert.NoError(t, err)
	<-watcher.ResultChan()

	result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
	assert.Error(t, err)
	assert.Equal(t, defaultErrRequeuePeriod, result.RequeueAfter)

	result, err = r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
	assert.NoError(t, err)
	assert.Equal(t, defaultRequeuePeriod, result.RequeueAfter)

	namespacedName := types.NamespacedName{Namespace: "default", Name: "test"}
	actualRecordingRuleGroupSetCRD := &coralogixv1alpha1.RecordingRuleGroupSet{}
	err = r.Client.Get(ctx, namespacedName, actualRecordingRuleGroupSetCRD)
	assert.NoError(t, err)

	err = r.Client.Delete(ctx, actualRecordingRuleGroupSetCRD)
	<-watcher.ResultChan()
	r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "default", Name: "test"}})
}

// createRecordingRuleGroupClientSimpleMock creates a simple mock for RecordingRuleGroupsClientInterface which returns a single recording rule group.
func createRecordingRuleGroupClientSimpleMock(mockCtrl *gomock.Controller) clientset.RecordingRulesGroupsClientInterface {
	mockRecordingRuleGroupsClient := mock_clientset.NewMockRecordingRulesGroupsClientInterface(mockCtrl)

	var recordingRuleGroupExist bool

	mockRecordingRuleGroupsClient.EXPECT().
		CreateRecordingRuleGroupSet(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *rrg.CreateRuleGroupSet) (*rrg.CreateRuleGroupSetResult, error) {
		recordingRuleGroupExist = true
		return &rrg.CreateRuleGroupSetResult{Id: "id1"}, nil
	}).AnyTimes()

	mockRecordingRuleGroupsClient.EXPECT().
		GetRecordingRuleGroupSet(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, req *rrg.FetchRuleGroupSet) (*rrg.OutRuleGroupSet, error) {
		if recordingRuleGroupExist {
			return recordingRuleGroupSetBackendSchema, nil
		}
		return nil, errors.NewNotFound(schema.GroupResource{}, "id1")
	}).AnyTimes()

	mockRecordingRuleGroupsClient.EXPECT().
		DeleteRecordingRuleGroupSet(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *rrg.DeleteRuleGroupSet) (*emptypb.Empty, error) {
		if recordingRuleGroupExist {
			recordingRuleGroupExist = false
			return &emptypb.Empty{}, nil
		}
		return nil, errors.NewNotFound(schema.GroupResource{}, "id1")
	}).AnyTimes()

	return mockRecordingRuleGroupsClient
}

// createRecordingRuleGroupClientMockWith5XXStatusError creates a simple mock for RecordingRuleGroupsClientInterface which first returns 5xx status error, then returns a single recording rule group.
func createRecordingRuleGroupClientMockWith5XXStatusError(mockCtrl *gomock.Controller) clientset.RecordingRulesGroupsClientInterface {
	mockRecordingRuleGroupsClient := mock_clientset.NewMockRecordingRulesGroupsClientInterface(mockCtrl)

	var recordingRuleGroupExist bool
	var wasCalled bool

	mockRecordingRuleGroupsClient.EXPECT().
		CreateRecordingRuleGroupSet(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, _ *rrg.CreateRuleGroupSet) (*rrg.CreateRuleGroupSetResult, error) {
		if !wasCalled {
			wasCalled = true
			return nil, errors.NewInternalError(fmt.Errorf("internal error"))
		}
		recordingRuleGroupExist = true
		return &rrg.CreateRuleGroupSetResult{Id: "id1"}, nil
	}).AnyTimes()

	mockRecordingRuleGroupsClient.EXPECT().
		GetRecordingRuleGroupSet(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, req *rrg.FetchRuleGroupSet) (*rrg.OutRuleGroupSet, error) {
		if recordingRuleGroupExist {
			return recordingRuleGroupSetBackendSchema, nil
		}
		return nil, errors.NewNotFound(schema.GroupResource{}, "id1")
	}).AnyTimes()

	mockRecordingRuleGroupsClient.EXPECT().
		DeleteRecordingRuleGroupSet(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, req *rrg.DeleteRuleGroupSet) (*emptypb.Empty, error) {
		if recordingRuleGroupExist {
			recordingRuleGroupExist = false
			return &emptypb.Empty{}, nil
		}
		return nil, errors.NewNotFound(schema.GroupResource{}, "id1")
	}).AnyTimes()

	return mockRecordingRuleGroupsClient
}

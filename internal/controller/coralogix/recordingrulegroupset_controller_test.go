// Copyright 2024 Coralogix Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package coralogix

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/controller/mock_clientset"
)

func setupRecordingRuleReconciler(t *testing.T, ctx context.Context, recordingRuleGroupSetClient *mock_clientset.MockRecordingRulesGroupsClientInterface) (RecordingRuleGroupSetReconciler, watch.Interface) {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	scheme := runtime.NewScheme()
	utilruntime.Must(coralogixv1alpha1.AddToScheme(scheme))

	mgr, _ := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:  scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	})

	go mgr.GetCache().Start(ctx)

	mgr.GetCache().WaitForCacheSync(ctx)
	withWatch, err := client.NewWithWatch(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
	})

	assert.NoError(t, err)
	r := RecordingRuleGroupSetReconciler{
		Client:                      withWatch,
		Scheme:                      mgr.GetScheme(),
		RecordingRuleGroupSetClient: recordingRuleGroupSetClient,
	}
	r.SetupWithManager(mgr)

	watcher, _ := r.Client.(client.WithWatch).Watch(ctx, &coralogixv1alpha1.RecordingRuleGroupSetList{})
	return r, watcher
}

type PrepareRecordingRulesParams struct {
	ctx                 context.Context
	recordingRuleClient *mock_clientset.MockRecordingRulesGroupsClientInterface
}

func TestRecordingRuleCreation(t *testing.T) {
	tests := []struct {
		name          string
		params        func(params PrepareRecordingRulesParams)
		recordingRule coralogixv1alpha1.RecordingRuleGroupSet
		shouldFail    bool
	}{
		{
			name:       "Recording rule creation success",
			shouldFail: false,
			params: func(params PrepareRecordingRulesParams) {
				params.recordingRuleClient.EXPECT().Create(params.ctx, gomock.Any()).Return(&cxsdk.CreateRuleGroupSetResponse{Id: "id1"}, nil)
			},
			recordingRule: coralogixv1alpha1.RecordingRuleGroupSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "recording-rule-creation-success",
					Namespace: "default",
				},
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			recordingRuleClient := mock_clientset.NewMockRecordingRulesGroupsClientInterface(controller)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.params != nil {
				tt.params(PrepareRecordingRulesParams{
					ctx:                 ctx,
					recordingRuleClient: recordingRuleClient,
				})
			}

			reconciler, watcher := setupRecordingRuleReconciler(t, ctx, recordingRuleClient)

			err := reconciler.Client.Create(ctx, &tt.recordingRule)

			assert.NoError(t, err)

			<-watcher.ResultChan()

			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.recordingRule.Namespace,
					Name:      tt.recordingRule.Name,
				},
			})

			if tt.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRecordingRuleUpdate(t *testing.T) {
	tests := []struct {
		name          string
		params        func(params PrepareRecordingRulesParams)
		recordingRule coralogixv1alpha1.RecordingRuleGroupSet
		shouldFail    bool
	}{
		{
			name:       "Recording rule update success",
			shouldFail: false,
			params: func(params PrepareRecordingRulesParams) {
				params.recordingRuleClient.EXPECT().Create(params.ctx, gomock.Any()).Return(&cxsdk.CreateRuleGroupSetResponse{Id: "id1"}, nil)
				params.recordingRuleClient.EXPECT().Get(params.ctx, gomock.Any()).Return(&cxsdk.GetRuleGroupSetResponse{
					Id: "id1",
					Groups: []*cxsdk.OutRuleGroup{
						{
							Name:     "name",
							Interval: pointer.Uint32(60),
							Limit:    pointer.Uint64(100),
							Rules: []*cxsdk.OutRule{
								{
									Record: "record",
									Expr:   "vector(1)",
									Labels: map[string]string{"key": "value"},
								},
							},
						},
					},
				}, nil)
				params.recordingRuleClient.EXPECT().Update(params.ctx, gomock.Any()).Return(&emptypb.Empty{}, nil)
			},
			recordingRule: coralogixv1alpha1.RecordingRuleGroupSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "recording-rule-update-success",
					Namespace: "default",
				},
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			defer controller.Finish()

			recordingRuleClient := mock_clientset.NewMockRecordingRulesGroupsClientInterface(controller)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.params != nil {
				tt.params(PrepareRecordingRulesParams{
					ctx:                 ctx,
					recordingRuleClient: recordingRuleClient,
				})
			}

			reconciler, watcher := setupRecordingRuleReconciler(t, ctx, recordingRuleClient)

			err := reconciler.Client.Create(ctx, &tt.recordingRule)

			assert.NoError(t, err)

			<-watcher.ResultChan()

			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.recordingRule.Namespace,
					Name:      tt.recordingRule.Name,
				},
			})

			assert.NoError(t, err)

			recordingRuleGroupSet := &coralogixv1alpha1.RecordingRuleGroupSet{}

			err = reconciler.Get(ctx, types.NamespacedName{
				Namespace: tt.recordingRule.Namespace,
				Name:      tt.recordingRule.Name,
			}, recordingRuleGroupSet)

			assert.NoError(t, err)

			err = reconciler.Client.Update(ctx, recordingRuleGroupSet)
			assert.NoError(t, err)

			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.recordingRule.Namespace,
					Name:      tt.recordingRule.Name,
				},
			})

			if tt.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

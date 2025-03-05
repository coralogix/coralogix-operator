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

package v1alpha1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/types/known/wrapperspb"
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

	"github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/config"
	"github.com/coralogix/coralogix-operator/internal/controller/mock_clientset"
)

func setupOutboundWebhooksReconciler(t *testing.T, ctx context.Context, outboundWebhooksClient *mock_clientset.MockOutboundWebhooksClientInterface) (OutboundWebhookReconciler, watch.Interface) {
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

	err = config.InitConfig(withWatch, mgr.GetScheme(), "", "", "")
	assert.NoError(t, err)

	r := OutboundWebhookReconciler{
		OutboundWebhooksClient: outboundWebhooksClient,
	}
	r.SetupWithManager(mgr)

	watcher, _ := withWatch.Watch(ctx, &v1alpha1.OutboundWebhookList{})
	return r, watcher
}

type PrepareOutboundWebhooksParams struct {
	ctx                    context.Context
	outboundWebhooksClient *mock_clientset.MockOutboundWebhooksClientInterface
}

func TestOutboundWebhooksCreation(t *testing.T) {
	tests := []struct {
		name            string
		params          func(params PrepareOutboundWebhooksParams)
		outboundWebhook v1alpha1.OutboundWebhook
		shouldFail      bool
	}{
		{
			name:       "outbound-webhook creation success",
			shouldFail: false,
			params: func(params PrepareOutboundWebhooksParams) {
				params.outboundWebhooksClient.EXPECT().Create(params.ctx, gomock.Any()).Return(&cxsdk.CreateOutgoingWebhookResponse{Id: wrapperspb.String("id")}, nil)
				params.outboundWebhooksClient.EXPECT().Get(params.ctx, gomock.Any()).Return(&cxsdk.GetOutgoingWebhookResponse{
					Webhook: &cxsdk.OutgoingWebhook{
						Id:   wrapperspb.String("id"),
						Name: wrapperspb.String("name"),
						Type: cxsdk.WebhookTypeGeneric,
						Url:  wrapperspb.String("url"),
						Config: &cxsdk.GenericWebhook{
							GenericWebhook: &cxsdk.GenericWebhookConfig{
								Uuid:    wrapperspb.String("uuid"),
								Method:  cxsdk.GenericWebhookConfigGet,
								Headers: map[string]string{"key": "value"},
								Payload: wrapperspb.String("payload"),
							},
						},
					},
				}, nil)
			},
			outboundWebhook: v1alpha1.OutboundWebhook{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "outbound-webhook-creation-success",
					Namespace: "default",
				},
				Spec: v1alpha1.OutboundWebhookSpec{
					Name: "name",
					OutboundWebhookType: v1alpha1.OutboundWebhookType{
						GenericWebhook: &v1alpha1.GenericWebhook{
							Url:     "url",
							Method:  "Get",
							Headers: map[string]string{"key": "value"},
							Payload: pointer.String("payload"),
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

			outboundWebhooksClient := mock_clientset.NewMockOutboundWebhooksClientInterface(controller)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.params != nil {
				tt.params(PrepareOutboundWebhooksParams{
					ctx:                    ctx,
					outboundWebhooksClient: outboundWebhooksClient,
				})
			}

			reconciler, watcher := setupOutboundWebhooksReconciler(t, ctx, outboundWebhooksClient)

			err := config.GetConfig().Client.Create(ctx, &tt.outboundWebhook)

			assert.NoError(t, err)

			<-watcher.ResultChan()

			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.outboundWebhook.Namespace,
					Name:      tt.outboundWebhook.Name,
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

func TestOutboundWebhookUpdate(t *testing.T) {
	tests := []struct {
		name            string
		params          func(params PrepareOutboundWebhooksParams)
		outboundWebhook v1alpha1.OutboundWebhook
		updatedWebhook  v1alpha1.OutboundWebhook
		shouldFail      bool
	}{
		{
			name:       "outbound-webhook update success",
			shouldFail: false,
			params: func(params PrepareOutboundWebhooksParams) {
				params.outboundWebhooksClient.EXPECT().Create(params.ctx, gomock.Any()).Return(&cxsdk.CreateOutgoingWebhookResponse{Id: wrapperspb.String("id")}, nil)
				params.outboundWebhooksClient.EXPECT().Get(params.ctx, gomock.Any()).Return(&cxsdk.GetOutgoingWebhookResponse{
					Webhook: &cxsdk.OutgoingWebhook{
						Id:   wrapperspb.String("id"),
						Name: wrapperspb.String("name"),
						Type: cxsdk.WebhookTypeGeneric,
						Url:  wrapperspb.String("url"),
						Config: &cxsdk.GenericWebhook{
							GenericWebhook: &cxsdk.GenericWebhookConfig{
								Uuid:    wrapperspb.String("uuid"),
								Method:  cxsdk.GenericWebhookConfigGet,
								Headers: map[string]string{"key": "value"},
								Payload: wrapperspb.String("payload"),
							},
						},
					},
				}, nil)
				params.outboundWebhooksClient.EXPECT().Update(params.ctx, gomock.Any()).Return(&cxsdk.UpdateOutgoingWebhookResponse{}, nil)
			},
			outboundWebhook: v1alpha1.OutboundWebhook{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "outbound-webhook-update-success",
					Namespace: "default",
				},
				Spec: v1alpha1.OutboundWebhookSpec{
					Name: "name",
					OutboundWebhookType: v1alpha1.OutboundWebhookType{
						GenericWebhook: &v1alpha1.GenericWebhook{
							Url:     "url",
							Method:  "Get",
							Headers: map[string]string{"key": "value"},
							Payload: pointer.String("payload"),
						},
					},
				},
			},
			updatedWebhook: v1alpha1.OutboundWebhook{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "outbound-webhook-update-success",
					Namespace: "default",
				},
				Spec: v1alpha1.OutboundWebhookSpec{
					Name: "updated-name",
					OutboundWebhookType: v1alpha1.OutboundWebhookType{
						GenericWebhook: &v1alpha1.GenericWebhook{
							Url:     "updated-url",
							Method:  "Post",
							Headers: map[string]string{"updated-key": "updated-value"},
							Payload: pointer.String("updated-payload"),
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

			outboundWebhookClient := mock_clientset.NewMockOutboundWebhooksClientInterface(controller)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.params != nil {
				tt.params(PrepareOutboundWebhooksParams{
					ctx:                    ctx,
					outboundWebhooksClient: outboundWebhookClient,
				})
			}

			reconciler, watcher := setupOutboundWebhooksReconciler(t, ctx, outboundWebhookClient)

			err := config.GetConfig().Client.Create(ctx, &tt.outboundWebhook)

			assert.NoError(t, err)

			<-watcher.ResultChan()

			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.outboundWebhook.Namespace,
					Name:      tt.outboundWebhook.Name,
				},
			})

			assert.NoError(t, err)

			outboundWebhook := &v1alpha1.OutboundWebhook{}

			err = config.GetConfig().Client.Get(ctx, types.NamespacedName{
				Namespace: tt.outboundWebhook.Namespace,
				Name:      tt.outboundWebhook.Name,
			}, outboundWebhook)

			assert.NoError(t, err)

			tt.updatedWebhook.ObjectMeta.ResourceVersion = outboundWebhook.ObjectMeta.ResourceVersion
			err = config.GetConfig().Client.Update(ctx, &tt.updatedWebhook)
			assert.NoError(t, err)

			<-watcher.ResultChan()

			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.updatedWebhook.Namespace,
					Name:      tt.updatedWebhook.Name,
				},
			})

			outboundWebhook = &v1alpha1.OutboundWebhook{}
			err = config.GetConfig().Client.Get(ctx, types.NamespacedName{
				Namespace: tt.updatedWebhook.Namespace,
				Name:      tt.updatedWebhook.Name,
			}, outboundWebhook)

			if tt.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOutboundWebhookDeletion(t *testing.T) {
	tests := []struct {
		name            string
		params          func(params PrepareOutboundWebhooksParams)
		outboundWebhook v1alpha1.OutboundWebhook
		shouldFail      bool
	}{
		{
			name:       "outbound-webhook deletion success",
			shouldFail: false,
			params: func(params PrepareOutboundWebhooksParams) {
				params.outboundWebhooksClient.EXPECT().Create(params.ctx, gomock.Any()).Return(&cxsdk.CreateOutgoingWebhookResponse{Id: wrapperspb.String("id")}, nil)
				params.outboundWebhooksClient.EXPECT().Get(params.ctx, gomock.Any()).Return(&cxsdk.GetOutgoingWebhookResponse{
					Webhook: &cxsdk.OutgoingWebhook{
						Id:   wrapperspb.String("id"),
						Name: wrapperspb.String("name"),
						Type: cxsdk.WebhookTypeGeneric,
						Url:  wrapperspb.String("url"),
						Config: &cxsdk.GenericWebhook{
							GenericWebhook: &cxsdk.GenericWebhookConfig{
								Uuid:    wrapperspb.String("uuid"),
								Method:  cxsdk.GenericWebhookConfigGet,
								Headers: map[string]string{"key": "value"},
								Payload: wrapperspb.String("payload"),
							},
						},
					},
				}, nil)
				params.outboundWebhooksClient.EXPECT().Delete(params.ctx, gomock.Any()).Return(&cxsdk.DeleteOutgoingWebhookResponse{}, nil)
			},
			outboundWebhook: v1alpha1.OutboundWebhook{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "outbound-webhook-deletion-success",
					Namespace: "default",
				},
				Spec: v1alpha1.OutboundWebhookSpec{
					Name: "name",
					OutboundWebhookType: v1alpha1.OutboundWebhookType{
						GenericWebhook: &v1alpha1.GenericWebhook{
							Url:     "url",
							Method:  "Get",
							Headers: map[string]string{"key": "value"},
							Payload: pointer.String("payload"),
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

			outboundWebhooksClient := mock_clientset.NewMockOutboundWebhooksClientInterface(controller)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.params != nil {
				tt.params(PrepareOutboundWebhooksParams{
					ctx:                    ctx,
					outboundWebhooksClient: outboundWebhooksClient,
				})
			}

			reconciler, watcher := setupOutboundWebhooksReconciler(t, ctx, outboundWebhooksClient)

			err := config.GetConfig().Client.Create(ctx, &tt.outboundWebhook)

			assert.NoError(t, err)

			<-watcher.ResultChan()

			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.outboundWebhook.Namespace,
					Name:      tt.outboundWebhook.Name,
				},
			})

			err = config.GetConfig().Client.Delete(ctx, &tt.outboundWebhook)

			assert.NoError(t, err)

			<-watcher.ResultChan()

			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.outboundWebhook.Namespace,
					Name:      tt.outboundWebhook.Name,
				},
			})

			assert.NoError(t, err)
		})
	}
}

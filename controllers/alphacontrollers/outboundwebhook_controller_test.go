package alphacontrollers

import (
	"context"
	"testing"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	ow "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/outbound-webhooks"
	"github.com/coralogix/coralogix-operator/controllers/mock_clientset"
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
)

func setupOutboundWebhooksReconciler(t *testing.T, ctx context.Context, outboundWebhooksClient *mock_clientset.MockOutboundWebhooksClientInterface) (OutboundWebhookReconciler, watch.Interface) {
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	scheme := runtime.NewScheme()
	utilruntime.Must(coralogixv1alpha1.AddToScheme(scheme))

	mgr, _ := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: "0",
	})

	go mgr.GetCache().Start(ctx)

	mgr.GetCache().WaitForCacheSync(ctx)
	withWatch, err := client.NewWithWatch(mgr.GetConfig(), client.Options{
		Scheme: mgr.GetScheme(),
	})

	assert.NoError(t, err)
	r := OutboundWebhookReconciler{
		Client:                 withWatch,
		Scheme:                 mgr.GetScheme(),
		OutboundWebhooksClient: outboundWebhooksClient,
	}
	r.SetupWithManager(mgr)

	watcher, _ := r.Client.(client.WithWatch).Watch(ctx, &coralogixv1alpha1.OutboundWebhookList{})
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
		outboundWebhook coralogixv1alpha1.OutboundWebhook
		shouldFail      bool
	}{
		{
			name:       "outbound-webhook creation success",
			shouldFail: false,
			params: func(params PrepareOutboundWebhooksParams) {
				params.outboundWebhooksClient.EXPECT().CreateOutboundWebhook(params.ctx, gomock.Any()).Return(&ow.CreateOutgoingWebhookResponse{Id: wrapperspb.String("id")}, nil)
				params.outboundWebhooksClient.EXPECT().GetOutboundWebhook(params.ctx, gomock.Any()).Return(&ow.GetOutgoingWebhookResponse{
					Webhook: &ow.OutgoingWebhook{
						Id:   wrapperspb.String("id"),
						Name: wrapperspb.String("name"),
						Type: ow.WebhookType_GENERIC,
						Url:  wrapperspb.String("url"),
						Config: &ow.OutgoingWebhook_GenericWebhook{
							GenericWebhook: &ow.GenericWebhookConfig{
								Uuid:    wrapperspb.String("uuid"),
								Method:  ow.GenericWebhookConfig_GET,
								Headers: map[string]string{"key": "value"},
								Payload: wrapperspb.String("payload"),
							},
						},
					},
				}, nil)
			},
			outboundWebhook: coralogixv1alpha1.OutboundWebhook{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "outbound-webhook-creation-success",
					Namespace: "default",
				},
				Spec: coralogixv1alpha1.OutboundWebhookSpec{
					Name: "name",
					OutboundWebhookType: coralogixv1alpha1.OutboundWebhookType{
						GenericWebhook: &coralogixv1alpha1.GenericWebhook{
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

			err := reconciler.Client.Create(ctx, &tt.outboundWebhook)

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
		outboundWebhook coralogixv1alpha1.OutboundWebhook
		shouldFail      bool
	}{
		{
			name:       "outbound-webhook update success",
			shouldFail: false,
			params: func(params PrepareOutboundWebhooksParams) {
				params.outboundWebhooksClient.EXPECT().CreateOutboundWebhook(params.ctx, gomock.Any()).Return(&ow.CreateOutgoingWebhookResponse{Id: wrapperspb.String("id")}, nil)
				params.outboundWebhooksClient.EXPECT().GetOutboundWebhook(params.ctx, gomock.Any()).Return(&ow.GetOutgoingWebhookResponse{
					Webhook: &ow.OutgoingWebhook{
						Id:   wrapperspb.String("id"),
						Name: wrapperspb.String("name"),
						Type: ow.WebhookType_GENERIC,
						Url:  wrapperspb.String("url"),
						Config: &ow.OutgoingWebhook_GenericWebhook{
							GenericWebhook: &ow.GenericWebhookConfig{
								Uuid:    wrapperspb.String("uuid"),
								Method:  ow.GenericWebhookConfig_GET,
								Headers: map[string]string{"key": "value"},
								Payload: wrapperspb.String("payload"),
							},
						},
					},
				}, nil)
				params.outboundWebhooksClient.EXPECT().UpdateOutboundWebhook(params.ctx, gomock.Any()).Return(&ow.UpdateOutgoingWebhookResponse{}, nil)
			},
			outboundWebhook: coralogixv1alpha1.OutboundWebhook{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "outbound-webhook-update-success",
					Namespace: "default",
				},
				Spec: coralogixv1alpha1.OutboundWebhookSpec{
					Name: "name",
					OutboundWebhookType: coralogixv1alpha1.OutboundWebhookType{
						GenericWebhook: &coralogixv1alpha1.GenericWebhook{
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

			err := reconciler.Client.Create(ctx, &tt.outboundWebhook)

			assert.NoError(t, err)

			<-watcher.ResultChan()

			_, err = reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Namespace: tt.outboundWebhook.Namespace,
					Name:      tt.outboundWebhook.Name,
				},
			})

			assert.NoError(t, err)

			outboundWebhook := &coralogixv1alpha1.OutboundWebhook{}

			err = reconciler.Get(ctx, types.NamespacedName{
				Namespace: tt.outboundWebhook.Namespace,
				Name:      tt.outboundWebhook.Name,
			}, outboundWebhook)

			assert.NoError(t, err)

			err = reconciler.Client.Update(ctx, outboundWebhook)
			assert.NoError(t, err)

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

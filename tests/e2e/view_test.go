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

package e2e

import (
	"context"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/utils"
)

var _ = Describe("View", Ordered, func() {
	var (
		k8sClient   client.Client
		viewsClient *cxsdk.ViewsClient
		view        *coralogixv1alpha1.View
		viewFolder  *coralogixv1alpha1.ViewFolder
		viewID      int32

		viewName       = fmt.Sprintf("e2e-view-%d", time.Now().Unix())
		viewFolderName = fmt.Sprintf("e2e-view-folder-%d", time.Now().Unix())
	)

	BeforeEach(func() {
		k8sClient = ClientsInstance.GetControllerRuntimeClient()
		viewsClient = ClientsInstance.GetCoralogixClientSet().Views()
		viewFolder = getSampleViewFolder(viewFolderName, testNamespace)
		view = getSampleView(viewName, viewFolderName, testNamespace)
	})

	It("Should create the View and ViewFolder successfully", func(ctx context.Context) {
		By("Creating ViewFolder")
		Expect(k8sClient.Create(ctx, viewFolder)).To(Succeed())

		By("Waiting for ViewFolder ID to be populated")
		Eventually(func(g Gomega) *string {
			viewFolder := &coralogixv1alpha1.ViewFolder{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: viewFolderName, Namespace: testNamespace}, viewFolder)
			g.Expect(err).To(Succeed())
			return viewFolder.Status.ID
		}, time.Minute, time.Second).Should(Not(BeNil()))

		By("Creating View")
		Expect(k8sClient.Create(ctx, view)).To(Succeed())

		By("Waiting for View ID to be populated")
		Eventually(func(g Gomega) error {
			view := &coralogixv1alpha1.View{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: viewName, Namespace: testNamespace}, view)).To(Succeed())
			g.Expect(meta.IsStatusConditionTrue(view.Status.Conditions, utils.ConditionTypeRemoteSynced)).To(BeTrue())
			g.Expect(view.Status.PrintableStatus).To(Equal("RemoteSynced"))
			if view.Status.ID == nil {
				return fmt.Errorf("view ID not set")
			}

			id, err := strconv.Atoi(*view.Status.ID)
			Expect(err).ToNot(HaveOccurred())
			viewID = int32(id)
			return nil
		}, time.Minute, time.Second).Should(Succeed())

		By("Verifying the View exists in the backend")
		Eventually(func() error {
			_, err := viewsClient.Get(context.Background(), &cxsdk.GetViewRequest{Id: wrapperspb.Int32(viewID)})
			return err
		}, time.Minute, time.Second).Should(Succeed())
	})

	It("Should update the View successfully", func(ctx context.Context) {
		By("Patching the View")
		newViewName := fmt.Sprintf("e2e-view-updated-%d", time.Now().Unix())
		modifiedView := view.DeepCopy()
		modifiedView.Spec.Name = newViewName
		Expect(k8sClient.Patch(ctx, modifiedView, client.MergeFrom(view))).To(Succeed())

		By("Verifying the View was updated in the backend")
		Eventually(func() string {
			fetched, err := viewsClient.Get(ctx, &cxsdk.GetViewRequest{Id: wrapperspb.Int32(viewID)})
			Expect(err).To(Succeed())
			return fetched.View.Name.GetValue()
		}, time.Minute, time.Second).Should(Equal(newViewName))
	})

	It("Should delete the View successfully", func(ctx context.Context) {
		By("Deleting View")
		Expect(k8sClient.Delete(ctx, view)).To(Succeed())

		By("Verifying the View was deleted in the backend")
		Eventually(func() codes.Code {
			_, err := viewsClient.Get(ctx, &cxsdk.GetViewRequest{Id: wrapperspb.Int32(viewID)})
			return cxsdk.Code(err)
		}, time.Minute, time.Second).Should(Equal(codes.NotFound))
	})

	It("Should deny a View creation with an invalid timeSelection", func(ctx context.Context) {
		By("Creating a view with both quickSelection and customSelection")
		view.Spec.TimeSelection.CustomSelection = &coralogixv1alpha1.CustomTimeSelection{
			FromTime: metav1.Time{
				Time: time.Now().Add(time.Hour),
			},
			ToTime: metav1.Time{
				Time: time.Now().Add(time.Hour * 2),
			},
		}

		err := k8sClient.Create(ctx, view)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Exactly one of quickSelection or customSelection must be set"))
	})
})

func getSampleViewFolder(name, namespace string) *coralogixv1alpha1.ViewFolder {
	return &coralogixv1alpha1.ViewFolder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coralogixv1alpha1.ViewFolderSpec{
			Name: name,
		},
	}
}

func getSampleView(name, folderName, namespace string) *coralogixv1alpha1.View {
	return &coralogixv1alpha1.View{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: coralogixv1alpha1.ViewSpec{
			Name: name,
			SearchQuery: &coralogixv1alpha1.SearchQuery{
				Query: "region:us-west-2",
			},
			TimeSelection: coralogixv1alpha1.TimeSelection{
				QuickSelection: &coralogixv1alpha1.QuickTimeSelection{
					Seconds: 900,
				},
			},
			Filters: coralogixv1alpha1.SelectedFilters{
				Filters: []coralogixv1alpha1.ViewFilter{
					{Name: "applicationName", SelectedValues: map[string]bool{
						"sample-app": true,
					}},
					{Name: "subsystemName", SelectedValues: map[string]bool{
						"sample-subsystem": true,
					}},
					{Name: "severity", SelectedValues: map[string]bool{
						"ERROR":   true,
						"WARNING": true,
					}},
				},
			},
			Folder: &coralogixv1alpha1.Folder{
				ResourceRef: &coralogixv1alpha1.ResourceRef{
					Name: folderName,
				},
			},
		},
	}
}

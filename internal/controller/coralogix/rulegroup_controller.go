/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package coralogix

import (
	"context"

	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cxsdk "github.com/coralogix/coralogix-management-sdk/go"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/api/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/internal/controller/clientset"
)

var ruleGroupFinalizerName = "rulegroup.coralogix.com/finalizer"

// RuleGroupReconciler reconciles a RuleGroup object
type RuleGroupReconciler struct {
	client.Client
	CoralogixClientSet clientset.ClientSetInterface
	Scheme             *runtime.Scheme
}

//+kubebuilder:rbac:groups=coralogix.com,resources=rulegroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=rulegroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=rulegroups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RuleGroup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *RuleGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	jsm := &jsonpb.Marshaler{
		EmitDefaults: true,
	}
	rulesGroupsClient := r.CoralogixClientSet.RuleGroups()

	//Get ruleGroupCRD
	ruleGroupCRD := &coralogixv1alpha1.RuleGroup{}

	if err := r.Client.Get(ctx, req.NamespacedName, ruleGroupCRD); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}

	// examine DeletionTimestamp to determine if object is under deletion
	if ruleGroupCRD.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(ruleGroupCRD, ruleGroupFinalizerName) {
			controllerutil.AddFinalizer(ruleGroupCRD, ruleGroupFinalizerName)
			if err := r.Update(ctx, ruleGroupCRD); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(ruleGroupCRD, ruleGroupFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if ruleGroupCRD.Status.ID == nil {
				controllerutil.RemoveFinalizer(ruleGroupCRD, ruleGroupFinalizerName)
				err := r.Update(ctx, ruleGroupCRD)
				return ctrl.Result{}, err
			}

			ruleGroupId := *ruleGroupCRD.Status.ID
			deleteRuleGroupReq := &cxsdk.DeleteRuleGroupRequest{GroupId: ruleGroupId}
			log.V(1).Info("Deleting Rule-Group", "Rule-Group ID", ruleGroupId)
			if _, err := rulesGroupsClient.Delete(ctx, deleteRuleGroupReq); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried unless it is deleted manually.
				log.Error(err, "Received an error while Deleting a Rule-Group", "Rule-Group ID", ruleGroupId)
				if status.Code(err) == codes.NotFound {
					controllerutil.RemoveFinalizer(ruleGroupCRD, ruleGroupFinalizerName)
					err := r.Update(ctx, ruleGroupCRD)
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, err
			}

			log.V(1).Info("Rule-Group was deleted", "Rule-Group ID", ruleGroupId)
			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(ruleGroupCRD, ruleGroupFinalizerName)
			if err := r.Update(ctx, ruleGroupCRD); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	var (
		notFound bool
		err      error
	)

	if id := ruleGroupCRD.Status.ID; id == nil {
		log.V(1).Info("ruleGroup wasn't created")
		notFound = true
	} else {
		_, err := rulesGroupsClient.Get(ctx, &cxsdk.GetRuleGroupRequest{GroupId: *id})
		switch {
		case status.Code(err) == codes.NotFound:
			log.V(1).Info("ruleGroup doesn't exist in Coralogix backend")
			notFound = true
		case err != nil:
			log.Error(err, "Received an error while getting RuleGroup")
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		}
	}

	if notFound {
		createRuleGroupReq := ruleGroupCRD.Spec.ExtractCreateRuleGroupRequest()
		jstr, _ := jsm.MarshalToString(createRuleGroupReq)
		log.V(1).Info("Creating Rule-Group", "ruleGroup", jstr)
		if createRuleGroupResp, err := rulesGroupsClient.Create(ctx, createRuleGroupReq); err == nil {
			jstr, _ := jsm.MarshalToString(createRuleGroupResp)
			log.V(1).Info("Rule-Group was updated", "ruleGroup", jstr)

			//To avoid a situation of the operator falling between the creation of the ruleGroup in coralogix and being saved in the cluster (something that would cause it to be created again and again), its id will be saved ASAP.
			id := createRuleGroupResp.GetRuleGroup().GetId().GetValue()
			ruleGroupCRD.Status = coralogixv1alpha1.RuleGroupStatus{ID: &id}
			if err := r.Status().Update(ctx, ruleGroupCRD); err != nil {
				log.Error(err, "Error on updating RecordingRuleGroupSet status", "Name", ruleGroupCRD.Name, "Namespace", ruleGroupCRD.Namespace)
				return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
			}

			status, err := flattenRuleGroup(createRuleGroupResp.GetRuleGroup())
			if err != nil {
				log.Error(err, "Error mapping coralogix API response", "Name", ruleGroupCRD.Name, "Namespace", ruleGroupCRD.Namespace)
				return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
			}
			ruleGroupCRD.Status = *status
			if err := r.Status().Update(ctx, ruleGroupCRD); err != nil {
				log.V(1).Error(err, "updating crd")
			}
			return ctrl.Result{}, nil
		} else {
			log.Error(err, "Received an error while creating a Rule-Group", "ruleGroup", createRuleGroupReq)
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		}
	} else if err != nil {
		log.Error(err, "Received an error while reading a Rule-Group", "ruleGroup ID", *ruleGroupCRD.Status.ID)
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}

	log.V(1).Info("Updating Rule-Group", "ruleGroup")
	updateRuleGroupReq := ruleGroupCRD.Spec.ExtractUpdateRuleGroupRequest(*ruleGroupCRD.Status.ID)
	updateRuleGroupResp, err := rulesGroupsClient.Update(ctx, updateRuleGroupReq)
	if err != nil {
		log.Error(err, "Received an error while updating a Rule-Group", "ruleGroup", updateRuleGroupReq)
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}
	jstr, _ := jsm.MarshalToString(updateRuleGroupResp)
	log.V(1).Info("Rule-Group was updated", "ruleGroup", jstr)

	return ctrl.Result{}, nil
}

func flattenRuleGroup(ruleGroup *cxsdk.RuleGroup) (*coralogixv1alpha1.RuleGroupStatus, error) {
	var status coralogixv1alpha1.RuleGroupStatus

	status.ID = new(string)
	*status.ID = ruleGroup.GetId().GetValue()

	return &status, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RuleGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.RuleGroup{}).
		Complete(r)
}

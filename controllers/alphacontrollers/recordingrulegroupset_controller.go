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

package alphacontrollers

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"
	"github.com/coralogix/coralogix-operator/controllers/clientset"
	rrg "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/recording-rules-groups/v2"

	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// RecordingRuleGroupSetReconciler reconciles a RecordingRuleGroupSet object
type RecordingRuleGroupSetReconciler struct {
	client.Client
	CoralogixClientSet *clientset.ClientSet
	Scheme             *runtime.Scheme
}

//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.com,resources=recordingrulegroupsets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RecordingRuleGroupSet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *RecordingRuleGroupSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	jsm := &jsonpb.Marshaler{
		//Indent: "\t",
	}
	rRGClient := r.CoralogixClientSet.RecordingRuleGroups()

	//Get ruleGroupSetRD
	ruleGroupSetCRD := &coralogixv1alpha1.RecordingRuleGroupSet{}
	if err := r.Client.Get(ctx, req.NamespacedName, ruleGroupSetCRD); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}

	// name of our custom finalizer
	myFinalizerName := "batch.tutorial.kubebuilder.io/finalizer"

	// examine DeletionTimestamp to determine if object is under deletion
	if ruleGroupSetCRD.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(ruleGroupSetCRD, myFinalizerName) {
			controllerutil.AddFinalizer(ruleGroupSetCRD, myFinalizerName)
			if err := r.Update(ctx, ruleGroupSetCRD); err != nil {
				log.Error(err, "Received an error while Updating a RecordingRuleGroupSet", "recordingRuleGroup Name", ruleGroupSetCRD.Name)
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(ruleGroupSetCRD, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if ruleGroupSetCRD.Status.ID == nil {
				controllerutil.RemoveFinalizer(ruleGroupSetCRD, myFinalizerName)
				err := r.Update(ctx, ruleGroupSetCRD)
				log.Error(err, "Received an error while Updating a RecordingRuleGroupSet", "recordingRuleGroup Name", ruleGroupSetCRD.Name)
				return ctrl.Result{}, err
			}

			id := *ruleGroupSetCRD.Status.ID
			deleteRRGReq := &rrg.DeleteRuleGroupSet{Id: id}
			log.V(1).Info("Deleting RecordingRuleGroupSet", "recordingRuleGroup ID", id)
			if _, err := rRGClient.DeleteRecordingRuleGroupSet(ctx, deleteRRGReq); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				log.Error(err, "Received an error while Deleting a RecordingRuleGroupSet", "recordingRuleGroup ID", id)
				return ctrl.Result{}, err
			}

			log.V(1).Info("RecordingRuleGroupSet was deleted", "RecordingRuleGroupSet ID", id)
			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(ruleGroupSetCRD, myFinalizerName)
			if err := r.Update(ctx, ruleGroupSetCRD); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	var notFount bool
	var err error
	var actualState coralogixv1alpha1.RecordingRuleGroupSetStatus
	if id := ruleGroupSetCRD.Status.ID; id == nil {
		log.V(1).Info("RecordingRuleGroupSet wasn't created in Coralogix backend")
		notFount = true
	} else if getRuleGroupSetResp, err := rRGClient.GetRecordingRuleGroupSet(ctx, &rrg.FetchRuleGroupSet{Id: *id}); status.Code(err) == codes.NotFound {
		log.V(1).Info("RecordingRuleGroupSet doesn't exist in Coralogix backend")
		notFount = true
	} else if err == nil {
		actualState = flattenRecordingRuleGroupSet(getRuleGroupSetResp)
	}

	if notFount {
		groups := ruleGroupSetCRD.Spec.ExtractRecordingRuleGroups()
		createRuleGroupReq := &rrg.CreateRuleGroupSet{Groups: groups}
		jstr, _ := jsm.MarshalToString(createRuleGroupReq)
		log.V(1).Info("Creating RecordingRuleGroupSet", "RecordingRuleGroupSet", jstr)
		if createRRGResp, err := rRGClient.CreateRecordingRuleGroupSet(ctx, createRuleGroupReq); err == nil {
			jstr, _ := jsm.MarshalToString(createRRGResp)
			log.V(1).Info("RecordingRuleGroupSet was updated", "RecordingRuleGroupSet", jstr)

			//To avoid a situation of the operator falling between the creation of the ruleGroup in coralogix and being saved in the cluster (something that would cause it to be created again and again), its id will be saved ASAP.
			id := createRRGResp.Id
			ruleGroupSetCRD.Status = coralogixv1alpha1.RecordingRuleGroupSetStatus{ID: &id}
			if err := r.Status().Update(ctx, ruleGroupSetCRD); err != nil {
				log.Error(err, "Error on updating RecordingRuleGroupSet status", "Name", ruleGroupSetCRD.Name, "Namespace", ruleGroupSetCRD.Namespace)
				return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
			}

			getRuleGroupReq := &rrg.FetchRuleGroupSet{Id: createRRGResp.Id}
			var getRRGResp *rrg.OutRuleGroupSet
			if getRRGResp, err = rRGClient.GetRecordingRuleGroupSet(ctx, getRuleGroupReq); err != nil || ruleGroupSetCRD == nil {
				log.Error(err, "Received an error while getting a RecordingRuleGroupSet", "RecordingRuleGroupSet", createRuleGroupReq)
				return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
			}
			ruleGroupSetCRD.Status = flattenRecordingRuleGroupSet(getRRGResp)
			if err := r.Status().Update(ctx, ruleGroupSetCRD); err != nil {
				log.V(1).Error(err, "updating crd")
			}
			return ctrl.Result{RequeueAfter: defaultRequeuePeriod}, nil
		} else {
			log.Error(err, "Received an error while creating a RecordingRuleGroupSet", "recordingRuleGroupSet", createRuleGroupReq)
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		}
	} else if err != nil {
		log.Error(err, "Received an error while reading a RecordingRuleGroupSet", "recordingRuleGroupSet ID", *ruleGroupSetCRD.Status.ID)
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}

	if equal, diff := ruleGroupSetCRD.Spec.DeepEqual(actualState); !equal {
		log.V(1).Info("Find diffs between spec and the actual state", "Diff", diff)
		id := *ruleGroupSetCRD.Status.ID
		groups := ruleGroupSetCRD.Spec.ExtractRecordingRuleGroups()
		updateRRGReq := &rrg.UpdateRuleGroupSet{Id: id, Groups: groups}
		if updateRRGResp, err := rRGClient.UpdateRecordingRuleGroupSet(ctx, updateRRGReq); err != nil {
			log.Error(err, "Received an error while updating a RecordingRuleGroupSet", "recordingRuleGroup", updateRRGReq)
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		} else {
			jstr, _ := jsm.MarshalToString(updateRRGResp)
			log.V(1).Info("RecordingRuleGroupSet was updated on backend", "recordingRuleGroup", jstr)
			var getRuleGroupResp *rrg.OutRuleGroupSet
			if getRuleGroupResp, err = rRGClient.GetRecordingRuleGroupSet(ctx, &rrg.FetchRuleGroupSet{Id: *ruleGroupSetCRD.Status.ID}); err != nil {
				log.Error(err, "Received an error while updating a RecordingRuleGroupSet", "recordingRuleGroup", updateRRGReq)
				return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
			}

			r.Client.Get(ctx, req.NamespacedName, ruleGroupSetCRD)
			ruleGroupSetCRD.Status = flattenRecordingRuleGroupSet(getRuleGroupResp)
			if err := r.Status().Update(ctx, ruleGroupSetCRD); err != nil {
				log.V(1).Error(err, "Error on updating RuleGroupSet crd")
				return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
			}
		}
	}

	return ctrl.Result{RequeueAfter: defaultRequeuePeriod}, nil
}

func flattenRecordingRuleGroupSet(set *rrg.OutRuleGroupSet) coralogixv1alpha1.RecordingRuleGroupSetStatus {
	id := new(string)
	*id = set.Id

	groups := make([]coralogixv1alpha1.RecordingRuleGroup, 0, len(set.Groups))
	for _, ruleGroup := range set.Groups {
		rg := flattenRecordingRuleGroup(ruleGroup)
		groups = append(groups, rg)
	}

	return coralogixv1alpha1.RecordingRuleGroupSetStatus{
		ID:     id,
		Groups: groups,
	}
}

func flattenRecordingRuleGroup(rRG *rrg.OutRuleGroup) coralogixv1alpha1.RecordingRuleGroup {
	interval := int32(*rRG.Interval)
	limit := int64(*rRG.Limit)
	rules := flattenRecordingRules(rRG.Rules)

	return coralogixv1alpha1.RecordingRuleGroup{
		Name:            rRG.Name,
		IntervalSeconds: interval,
		Limit:           limit,
		Rules:           rules,
	}
}

func flattenRecordingRules(rules []*rrg.OutRule) []coralogixv1alpha1.RecordingRule {
	result := make([]coralogixv1alpha1.RecordingRule, 0, len(rules))
	for _, r := range rules {
		rule := flattenRecordingRule(r)
		result = append(result, rule)
	}
	return result
}

func flattenRecordingRule(rule *rrg.OutRule) coralogixv1alpha1.RecordingRule {
	return coralogixv1alpha1.RecordingRule{
		Record: rule.Record,
		Expr:   rule.Expr,
		Labels: rule.Labels,
	}
}

func (r *RecordingRuleGroupSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.RecordingRuleGroupSet{}).
		Complete(r)
}

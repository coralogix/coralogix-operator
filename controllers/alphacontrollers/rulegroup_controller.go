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
	"fmt"

	"github.com/coralogix/coralogix-operator/controllers/clientset"
	rulesgroups "github.com/coralogix/coralogix-operator/controllers/clientset/grpc/rules-groups/v1"

	"github.com/golang/protobuf/jsonpb"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	coralogixv1alpha1 "github.com/coralogix/coralogix-operator/apis/coralogix/v1alpha1"

	"google.golang.org/grpc/status"
)

// RuleGroupReconciler reconciles a RuleGroup object
type RuleGroupReconciler struct {
	client.Client
	CoralogixClientSet *clientset.ClientSet
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

	myFinalizerName := "coralogix.com/rulegroup/finalizer"

	// examine DeletionTimestamp to determine if object is under deletion
	if ruleGroupCRD.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !controllerutil.ContainsFinalizer(ruleGroupCRD, myFinalizerName) {
			controllerutil.AddFinalizer(ruleGroupCRD, myFinalizerName)
			if err := r.Update(ctx, ruleGroupCRD); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(ruleGroupCRD, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if ruleGroupCRD.Status.ID == nil {
				controllerutil.RemoveFinalizer(ruleGroupCRD, myFinalizerName)
				err := r.Update(ctx, ruleGroupCRD)
				return ctrl.Result{}, err
			}

			ruleGroupId := *ruleGroupCRD.Status.ID
			deleteRuleGroupReq := &rulesgroups.DeleteRuleGroupRequest{GroupId: ruleGroupId}
			log.V(1).Info("Deleting Rule-Group", "Rule-Group ID", ruleGroupId)
			if _, err := rulesGroupsClient.DeleteRuleGroup(ctx, deleteRuleGroupReq); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried unless it is deleted manually.
				log.Error(err, "Received an error while Deleting a Rule-Group", "Rule-Group ID", ruleGroupId)
				if status.Code(err) == codes.NotFound {
					controllerutil.RemoveFinalizer(ruleGroupCRD, myFinalizerName)
					err := r.Update(ctx, ruleGroupCRD)
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, err
			}

			log.V(1).Info("Rule-Group was deleted", "Rule-Group ID", ruleGroupId)
			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(ruleGroupCRD, myFinalizerName)
			if err := r.Update(ctx, ruleGroupCRD); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	var (
		notFound    bool
		err         error
		actualState *coralogixv1alpha1.RuleGroupStatus
	)

	if id := ruleGroupCRD.Status.ID; id == nil {
		log.V(1).Info("ruleGroup wasn't created")
		notFound = true
	} else {
		getRuleGroupResp, err := rulesGroupsClient.GetRuleGroup(ctx, &rulesgroups.GetRuleGroupRequest{GroupId: *id})
		switch {
		case status.Code(err) == codes.NotFound:
			log.V(1).Info("ruleGroup doesn't exist in Coralogix backend")
			notFound = true
		case err != nil:
			log.Error(err, "Received an error while getting RuleGroup")
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		case err == nil:
			actualState, err = flattenRuleGroup(getRuleGroupResp.GetRuleGroup())
			if err != nil {
				log.Error(err, "Error mapping coralogix API response", "Name", ruleGroupCRD.Name, "Namespace", ruleGroupCRD.Namespace)
				return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
			}
		}
	}

	if notFound {
		createRuleGroupReq := ruleGroupCRD.Spec.ExtractCreateRuleGroupRequest()
		jstr, _ := jsm.MarshalToString(createRuleGroupReq)
		log.V(1).Info("Creating Rule-Group", "ruleGroup", jstr)
		if createRuleGroupResp, err := rulesGroupsClient.CreateRuleGroup(ctx, createRuleGroupReq); err == nil {
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
			return ctrl.Result{RequeueAfter: defaultRequeuePeriod}, nil
		} else {
			log.Error(err, "Received an error while creating a Rule-Group", "ruleGroup", createRuleGroupReq)
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		}
	} else if err != nil {
		log.Error(err, "Received an error while reading a Rule-Group", "ruleGroup ID", *ruleGroupCRD.Status.ID)
		return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
	}

	if equal, diff := ruleGroupCRD.Spec.DeepEqual(*actualState); !equal {
		log.V(1).Info("Find diffs between spec and the actual state", "Diff", diff)
		updateRuleGroupReq := ruleGroupCRD.Spec.ExtractUpdateRuleGroupRequest(*ruleGroupCRD.Status.ID)
		updateRuleGroupResp, err := rulesGroupsClient.UpdateRuleGroup(ctx, updateRuleGroupReq)
		if err != nil {
			log.Error(err, "Received an error while updating a Rule-Group", "ruleGroup", updateRuleGroupReq)
			return ctrl.Result{RequeueAfter: defaultErrRequeuePeriod}, err
		}
		jstr, _ := jsm.MarshalToString(updateRuleGroupResp)
		log.V(1).Info("Rule-Group was updated", "ruleGroup", jstr)
	}

	return ctrl.Result{RequeueAfter: defaultRequeuePeriod}, nil
}

func flattenRuleGroup(ruleGroup *rulesgroups.RuleGroup) (*coralogixv1alpha1.RuleGroupStatus, error) {
	var status coralogixv1alpha1.RuleGroupStatus

	status.ID = new(string)
	*status.ID = ruleGroup.GetId().GetValue()

	status.Name = ruleGroup.GetName().GetValue()

	status.Active = ruleGroup.GetEnabled().GetValue()

	var err error
	status.Applications, status.Subsystems, status.Severities, err = flattenRuleMatcher(ruleGroup.GetRuleMatchers())
	if err != nil {
		return nil, fmt.Errorf("flattenRuleGroup name: %s: %w", ruleGroup.GetName().GetValue(), err)
	}

	status.Description = ruleGroup.Description.GetValue()

	status.Order = new(int32)
	*status.Order = int32(ruleGroup.GetOrder().GetValue())

	status.Creator = ruleGroup.GetCreator().GetValue()

	status.Hidden = ruleGroup.GetHidden().GetValue()

	subGroups, err := flattenRuleSubGroups(ruleGroup.GetRuleSubgroups())
	if err != nil {
		return nil, fmt.Errorf("flattenRuleGroup name: %s: %w", ruleGroup.GetName().GetValue(), err)
	}
	status.RuleSubgroups = subGroups
	return &status, nil
}

func flattenRuleSubGroups(subgroups []*rulesgroups.RuleSubgroup) ([]coralogixv1alpha1.RuleSubGroup, error) {
	result := make([]coralogixv1alpha1.RuleSubGroup, 0, len(subgroups))
	for _, sg := range subgroups {
		subgroup, err := flattenRuleSubGroup(sg)
		if err != nil {
			return nil, fmt.Errorf("flattenRuleSubGroups subGroupId: %s: %w", sg.GetId().GetValue(), err)
		}
		result = append(result, subgroup)
	}
	return result, nil
}

func flattenRuleSubGroup(subGroup *rulesgroups.RuleSubgroup) (coralogixv1alpha1.RuleSubGroup, error) {
	var result coralogixv1alpha1.RuleSubGroup

	result.ID = new(string)
	*result.ID = subGroup.Id.GetValue()

	result.Active = subGroup.GetEnabled().GetValue()

	result.Order = new(int32)
	*result.Order = int32(subGroup.GetOrder().GetValue())

	rules, err := flattenRules(subGroup.Rules)
	if err != nil {
		return coralogixv1alpha1.RuleSubGroup{}, fmt.Errorf("flattenRuleSubGroup: %w", err)
	}
	result.Rules = rules
	return result, nil
}

func flattenRules(rules []*rulesgroups.Rule) ([]coralogixv1alpha1.Rule, error) {
	result := make([]coralogixv1alpha1.Rule, 0, len(rules))
	for _, r := range rules {
		rule, err := flattenRule(r)
		if err != nil {
			return nil, fmt.Errorf("flattenRules: %w", err)
		}
		result = append(result, rule)
	}
	return result, nil
}

func flattenRule(rule *rulesgroups.Rule) (coralogixv1alpha1.Rule, error) {
	var result coralogixv1alpha1.Rule
	result.Name = rule.GetName().GetValue()
	result.Active = rule.GetEnabled().GetValue()
	result.Description = rule.GetDescription().GetValue()

	switch ruleParams := rule.GetParameters().GetRuleParameters().(type) {
	case *rulesgroups.RuleParameters_ExtractParameters:
		extractParameters := ruleParams.ExtractParameters
		result.Extract = &coralogixv1alpha1.Extract{
			Regex:       extractParameters.GetRule().GetValue(),
			SourceField: rule.GetSourceField().GetValue(),
		}
	case *rulesgroups.RuleParameters_JsonExtractParameters:
		jsonExtractParameters := ruleParams.JsonExtractParameters
		result.JsonExtract = &coralogixv1alpha1.JsonExtract{
			JsonKey:          jsonExtractParameters.GetRule().GetValue(),
			DestinationField: coralogixv1alpha1.RulesProtoSeverityDestinationFieldToSchemaDestinationField[jsonExtractParameters.GetDestinationField()],
		}
	case *rulesgroups.RuleParameters_ReplaceParameters:
		replaceParameters := ruleParams.ReplaceParameters
		result.Replace = &coralogixv1alpha1.Replace{
			SourceField:       rule.GetSourceField().GetValue(),
			DestinationField:  replaceParameters.GetDestinationField().GetValue(),
			Regex:             replaceParameters.GetRule().GetValue(),
			ReplacementString: replaceParameters.GetReplaceNewVal().GetValue(),
		}
	case *rulesgroups.RuleParameters_ParseParameters:
		parseParameters := ruleParams.ParseParameters
		result.Parse = &coralogixv1alpha1.Parse{
			SourceField:      rule.GetSourceField().GetValue(),
			DestinationField: parseParameters.GetDestinationField().GetValue(),
			Regex:            parseParameters.GetRule().GetValue(),
		}
	case *rulesgroups.RuleParameters_AllowParameters:
		allowParameters := ruleParams.AllowParameters
		result.Block = &coralogixv1alpha1.Block{
			SourceField:               rule.GetSourceField().GetValue(),
			Regex:                     allowParameters.GetRule().GetValue(),
			KeepBlockedLogs:           allowParameters.GetKeepBlockedLogs().GetValue(),
			BlockingAllMatchingBlocks: false,
		}
	case *rulesgroups.RuleParameters_BlockParameters:
		blockParameters := ruleParams.BlockParameters
		result.Block = &coralogixv1alpha1.Block{
			SourceField:               rule.GetSourceField().GetValue(),
			Regex:                     blockParameters.GetRule().GetValue(),
			KeepBlockedLogs:           blockParameters.GetKeepBlockedLogs().GetValue(),
			BlockingAllMatchingBlocks: true,
		}
	case *rulesgroups.RuleParameters_ExtractTimestampParameters:
		extractTimestampParameters := ruleParams.ExtractTimestampParameters
		result.ExtractTimestamp = &coralogixv1alpha1.ExtractTimestamp{
			SourceField:         rule.GetSourceField().GetValue(),
			TimeFormat:          extractTimestampParameters.GetFormat().GetValue(),
			FieldFormatStandard: coralogixv1alpha1.RulesProtoFormatStandardToSchemaFormatStandard[extractTimestampParameters.GetStandard()],
		}
	case *rulesgroups.RuleParameters_RemoveFieldsParameters:
		removeFieldsParameters := ruleParams.RemoveFieldsParameters
		result.RemoveFields = &coralogixv1alpha1.RemoveFields{
			ExcludedFields: removeFieldsParameters.GetFields(),
		}
	case *rulesgroups.RuleParameters_JsonStringifyParameters:
		jsonStringifyParameters := ruleParams.JsonStringifyParameters
		result.JsonStringify = &coralogixv1alpha1.JsonStringify{
			SourceField:      rule.GetSourceField().GetValue(),
			DestinationField: jsonStringifyParameters.GetDestinationField().GetValue(),
			KeepSourceField:  !(jsonStringifyParameters.GetDeleteSource().GetValue()),
		}
	case *rulesgroups.RuleParameters_JsonParseParameters:
		jsonParseParameters := ruleParams.JsonParseParameters
		result.ParseJsonField = &coralogixv1alpha1.ParseJsonField{
			SourceField:          rule.GetSourceField().GetValue(),
			DestinationField:     jsonParseParameters.GetDestinationField().GetValue(),
			KeepSourceField:      !(jsonParseParameters.GetDeleteSource().GetValue()),
			KeepDestinationField: !(jsonParseParameters.GetOverrideDest().GetValue()),
		}
	default:
		return coralogixv1alpha1.Rule{}, fmt.Errorf("unexpected type %T for rule parameters, name: %s", ruleParams, rule.GetName().GetValue())
	}
	return result, nil
}

func flattenRuleMatcher(ruleMatchers []*rulesgroups.RuleMatcher) (applications, subsystems []string, severities []coralogixv1alpha1.RuleSeverity, err error) {
	for _, ruleMatcher := range ruleMatchers {
		switch matcher := ruleMatcher.Constraint.(type) {
		case *rulesgroups.RuleMatcher_ApplicationName:
			applications = append(applications, matcher.ApplicationName.GetValue().GetValue())
		case *rulesgroups.RuleMatcher_SubsystemName:
			subsystems = append(subsystems, matcher.SubsystemName.GetValue().GetValue())
		case *rulesgroups.RuleMatcher_Severity:
			severity := matcher.Severity.GetValue()
			severities = append(severities, coralogixv1alpha1.RulesProtoSeverityToSchemaSeverity[severity])
		default:
			return nil, nil, nil, fmt.Errorf("unexpected type %T for rule matcher", ruleMatcher)
		}
	}

	return
}

// SetupWithManager sets up the controller with the Manager.
func (r *RuleGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&coralogixv1alpha1.RuleGroup{}).
		Complete(r)
}

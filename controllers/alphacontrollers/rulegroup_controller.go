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

//+kubebuilder:rbac:groups=coralogix.coralogix,resources=rulegroups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=coralogix.coralogix,resources=rulegroups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=coralogix.coralogix,resources=rulegroups/finalizers,verbs=update

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
		//Indent: "\t",
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

	// name of our custom finalizer
	myFinalizerName := "batch.tutorial.kubebuilder.io/finalizer"

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
				// so that it can be retried
				log.Error(err, "Received an error while Deleting a Rule-Group", "Rule-Group ID", ruleGroupId)
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

	var notFount bool
	var err error
	var actualState *coralogixv1alpha1.RuleGroupStatus
	if id := ruleGroupCRD.Status.ID; id == nil {
		log.V(1).Info("ruleGroup wasn't created")
		notFount = true
	} else if getRuleGroupResp, err := rulesGroupsClient.GetRuleGroup(ctx, &rulesgroups.GetRuleGroupRequest{GroupId: *id}); status.Code(err) == codes.NotFound {
		log.V(1).Info("ruleGroup doesn't exist in Coralogix backend")
		notFount = true
	} else if err == nil {
		actualState = flattenRuleGroup(getRuleGroupResp.GetRuleGroup())
	}

	if notFount {
		createRuleGroupReq := ruleGroupCRD.Spec.ExtractCreateRuleGroupRequest()
		jstr, _ := jsm.MarshalToString(createRuleGroupReq)
		log.V(1).Info("Creating Rule-Group", "ruleGroup", jstr)
		if createRuleGroupResp, err := rulesGroupsClient.CreateRuleGroup(ctx, createRuleGroupReq); err == nil {
			jstr, _ := jsm.MarshalToString(createRuleGroupResp)
			log.V(1).Info("Rule-Group was updated", "ruleGroup", jstr)
			ruleGroupCRD.Status = *flattenRuleGroup(createRuleGroupResp.GetRuleGroup())
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

func flattenRuleGroup(ruleGroup *rulesgroups.RuleGroup) *coralogixv1alpha1.RuleGroupStatus {
	var status coralogixv1alpha1.RuleGroupStatus

	status.ID = new(string)
	*status.ID = ruleGroup.GetId().GetValue()

	status.Name = ruleGroup.GetName().GetValue()

	status.Active = ruleGroup.GetEnabled().GetValue()

	status.Applications, status.Subsystems, status.Severities = flattenRuleMatcher(ruleGroup.GetRuleMatchers())

	status.Description = ruleGroup.Description.GetValue()

	status.Order = new(int32)
	*status.Order = int32(ruleGroup.GetOrder().GetValue())

	status.Creator = ruleGroup.GetCreator().GetValue()

	status.Hidden = ruleGroup.GetHidden().GetValue()

	status.RuleSubgroups = flattenRuleSubGroups(ruleGroup.GetRuleSubgroups())

	return &status
}

func flattenRuleSubGroups(subgroups []*rulesgroups.RuleSubgroup) []coralogixv1alpha1.RuleSubGroup {
	result := make([]coralogixv1alpha1.RuleSubGroup, 0, len(subgroups))
	for _, sg := range subgroups {
		subgroup := flattenRuleSubGroup(sg)
		result = append(result, subgroup)
	}
	return result
}

func flattenRuleSubGroup(subGroup *rulesgroups.RuleSubgroup) coralogixv1alpha1.RuleSubGroup {
	var result coralogixv1alpha1.RuleSubGroup

	result.ID = new(string)
	*result.ID = subGroup.Id.GetValue()

	result.Active = subGroup.GetEnabled().GetValue()

	result.Order = new(int32)
	*result.Order = int32(subGroup.GetOrder().GetValue())

	result.Rules = flattenRules(subGroup.Rules)

	return result
}

func flattenRules(rules []*rulesgroups.Rule) []coralogixv1alpha1.Rule {
	result := make([]coralogixv1alpha1.Rule, 0, len(rules))
	for _, r := range rules {
		rule := flattenRule(r)
		result = append(result, rule)
	}
	return result
}

func flattenRule(rule *rulesgroups.Rule) coralogixv1alpha1.Rule {
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
		panic(fmt.Sprintf("unexpected type %T for r parameters", ruleParams))
	}
	return result
}

func flattenRuleMatcher(ruleMatchers []*rulesgroups.RuleMatcher) (applications, subsystems []string, severities []coralogixv1alpha1.RuleSeverity) {
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
			panic(fmt.Sprintf("unexpected type %T for rule matcher", ruleMatcher))
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

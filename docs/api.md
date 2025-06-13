# API Reference

Packages:

- [coralogix.com/v1beta1](#coralogixcomv1beta1)
- [coralogix.com/v1alpha1](#coralogixcomv1alpha1)

# coralogix.com/v1beta1

Resource Types:

- [Alert](#alert)




## Alert
<sup><sup>[↩ Parent](#coralogixcomv1beta1 )</sup></sup>






Alert is the Schema for the alerts API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1beta1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Alert</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspec">spec</a></b></td>
        <td>object</td>
        <td>
          AlertSpec defines the desired state of a Coralogix Alert. For more info check - https://coralogix.com/docs/getting-started-with-coralogix-alerts/.

Note that this is only for the latest version of the alerts API. If your account has been created before March 2025, make sure that your account has been migrated before using advanced features of alerts.

Added in v0.4.0<br/>
          <br/>
            <i>Validations</i>:<li>!has(self.alertType.logsImmediate) || !has(self.groupByKeys): groupByKeys is not supported for this alert type</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertstatus">status</a></b></td>
        <td>object</td>
        <td>
          AlertStatus defines the observed state of Alert<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec
<sup><sup>[↩ Parent](#alert)</sup></sup>



AlertSpec defines the desired state of a Coralogix Alert. For more info check - https://coralogix.com/docs/getting-started-with-coralogix-alerts/.

Note that this is only for the latest version of the alerts API. If your account has been created before March 2025, make sure that your account has been migrated before using advanced features of alerts.

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttype">alertType</a></b></td>
        <td>object</td>
        <td>
          Type of alert.<br/>
          <br/>
            <i>Validations</i>:<li>(has(self.logsImmediate) ? 1 : 0) + (has(self.logsThreshold) ? 1 : 0) + (has(self.logsRatioThreshold) ? 1 : 0) + (has(self.logsTimeRelativeThreshold) ? 1 : 0) + (has(self.metricThreshold) ? 1 : 0) + (has(self.tracingThreshold) ? 1 : 0) + (has(self.tracingImmediate) ? 1 : 0) + (has(self.flow) ? 1 : 0) + (has(self.logsAnomaly) ? 1 : 0) + (has(self.metricAnomaly) ? 1 : 0) + (has(self.logsNewValue) ? 1 : 0) + (has(self.logsUniqueCount) ? 1 : 0) == 1: Exactly one of logsImmediate, logsThreshold, logsRatioThreshold, logsTimeRelativeThreshold, metricThreshold, tracingThreshold, tracingImmediate, flow, logsAnomaly, metricAnomaly, logsNewValue or logsUniqueCount must be set</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the alert<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>priority</b></td>
        <td>enum</td>
        <td>
          Priority of the alert.<br/>
          <br/>
            <i>Enum</i>: p1, p2, p3, p4, p5<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description of the alert<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          Enable/disable the alert.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>entityLabels</b></td>
        <td>map[string]string</td>
        <td>
          Labels attached to the alert.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>groupByKeys</b></td>
        <td>[]string</td>
        <td>
          Grouping fields for multiple alerts.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecincidentssettings">incidentsSettings</a></b></td>
        <td>object</td>
        <td>
          Settings for the attached incidents.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroup">notificationGroup</a></b></td>
        <td>object</td>
        <td>
          Where notifications should be sent to.<br/>
          <br/>
            <i>Validations</i>:<li>!(has(self.destinations) && has(self.router)): At most one of Destinations or Router can be set.</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindex">notificationGroupExcess</a></b></td>
        <td>[]object</td>
        <td>
          Do not use.
Deprecated: Legacy field for when multiple notification groups were attached.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>phantomMode</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecschedule">schedule</a></b></td>
        <td>object</td>
        <td>
          Alert activity schedule. Will be activated all the time if not specified.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType
<sup><sup>[↩ Parent](#alertspec)</sup></sup>



Type of alert.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypeflow">flow</a></b></td>
        <td>object</td>
        <td>
          Flow alerts chaining multiple alerts together.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsanomaly">logsAnomaly</a></b></td>
        <td>object</td>
        <td>
          Anomaly alerts for logs.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsimmediate">logsImmediate</a></b></td>
        <td>object</td>
        <td>
          Immediate alerts for logs.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsnewvalue">logsNewValue</a></b></td>
        <td>object</td>
        <td>
          Alerts when a new log value appears.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsratiothreshold">logsRatioThreshold</a></b></td>
        <td>object</td>
        <td>
          Alerts for when a log exceeds a defined ratio.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsthreshold">logsThreshold</a></b></td>
        <td>object</td>
        <td>
          Alerts for when a log crosses a threshold.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogstimerelativethreshold">logsTimeRelativeThreshold</a></b></td>
        <td>object</td>
        <td>
          Alerts are sent when the number of logs matching a filter is more than or less than a threshold over a specific time window.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsuniquecount">logsUniqueCount</a></b></td>
        <td>object</td>
        <td>
          Alerts for unique count changes.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricanomaly">metricAnomaly</a></b></td>
        <td>object</td>
        <td>
          Anomaly alerts for metrics.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricthreshold">metricThreshold</a></b></td>
        <td>object</td>
        <td>
          Alerts for when a metric crosses a threshold.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingimmediate">tracingImmediate</a></b></td>
        <td>object</td>
        <td>
          Immediate alerts for traces.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingthreshold">tracingThreshold</a></b></td>
        <td>object</td>
        <td>
          Alerts for when traces crosses a threshold.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>



Flow alerts chaining multiple alerts together.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enforceSuppression</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypeflowstagesindex">stages</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index]
<sup><sup>[↩ Parent](#alertspecalerttypeflow)</sup></sup>



Stages to go through.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypeflowstagesindexflowstagestype">flowStagesType</a></b></td>
        <td>object</td>
        <td>
          Type of stage.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>timeframeMs</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>timeframeType</b></td>
        <td>enum</td>
        <td>
          Type of timeframe.<br/>
          <br/>
            <i>Enum</i>: unspecified, upTo<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index].flowStagesType
<sup><sup>[↩ Parent](#alertspecalerttypeflowstagesindex)</sup></sup>



Type of stage.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypeflowstagesindexflowstagestypegroupsindex">groups</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index].flowStagesType.groups[index]
<sup><sup>[↩ Parent](#alertspecalerttypeflowstagesindexflowstagestype)</sup></sup>



Flow stage grouping.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypeflowstagesindexflowstagestypegroupsindexalertdefsindex">alertDefs</a></b></td>
        <td>[]object</td>
        <td>
          Alerts to group.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>alertsOp</b></td>
        <td>enum</td>
        <td>
          Operation for the alert.<br/>
          <br/>
            <i>Enum</i>: and, or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>nextOp</b></td>
        <td>enum</td>
        <td>
          Link to the next alert.<br/>
          <br/>
            <i>Enum</i>: and, or<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index].flowStagesType.groups[index].alertDefs[index]
<sup><sup>[↩ Parent](#alertspecalerttypeflowstagesindexflowstagestypegroupsindex)</sup></sup>



Alert references.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypeflowstagesindexflowstagestypegroupsindexalertdefsindexalertref">alertRef</a></b></td>
        <td>object</td>
        <td>
          Reference for an alert, backend or Kubernetes resource<br/>
          <br/>
            <i>Validations</i>:<li>has(self.backendRef) != has(self.resourceRef): Exactly one of backendRef or resourceRef must be set</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>not</b></td>
        <td>boolean</td>
        <td>
          Inversion.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index].flowStagesType.groups[index].alertDefs[index].alertRef
<sup><sup>[↩ Parent](#alertspecalerttypeflowstagesindexflowstagestypegroupsindexalertdefsindex)</sup></sup>



Reference for an alert, backend or Kubernetes resource

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypeflowstagesindexflowstagestypegroupsindexalertdefsindexalertrefbackendref">backendRef</a></b></td>
        <td>object</td>
        <td>
          Coralogix id reference.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.id) != has(self.name): One of id or name is required</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypeflowstagesindexflowstagestypegroupsindexalertdefsindexalertrefresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          Kubernetes resource reference.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index].flowStagesType.groups[index].alertDefs[index].alertRef.backendRef
<sup><sup>[↩ Parent](#alertspecalerttypeflowstagesindexflowstagestypegroupsindexalertdefsindexalertref)</sup></sup>



Coralogix id reference.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          Alert ID.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the alert.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index].flowStagesType.groups[index].alertDefs[index].alertRef.resourceRef
<sup><sup>[↩ Parent](#alertspecalerttypeflowstagesindexflowstagestypegroupsindexalertdefsindexalertref)</sup></sup>



Kubernetes resource reference.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>



Anomaly alerts for logs.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsanomalyrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          Rules that match the alert to the data.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsanomalylogsfilter">logsFilter</a></b></td>
        <td>object</td>
        <td>
          Filter to filter the logs with.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>notificationPayloadFilter</b></td>
        <td>[]string</td>
        <td>
          Filter for the notification payload.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsanomaly)</sup></sup>



The rule to match the alert's conditions.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsanomalyrulesindexcondition">condition</a></b></td>
        <td>object</td>
        <td>
          Condition to match to.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypelogsanomalyrulesindex)</sup></sup>



Condition to match to.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>minimumThreshold</b></td>
        <td>int or string</td>
        <td>
          Minimum value<br/>
          <br/>
            <i>Default</i>: 0<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsanomalyrulesindexconditiontimewindow">timeWindow</a></b></td>
        <td>object</td>
        <td>
          Time window to evaluate.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly.rules[index].condition.timeWindow
<sup><sup>[↩ Parent](#alertspecalerttypelogsanomalyrulesindexcondition)</sup></sup>



Time window to evaluate.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>specificValue</b></td>
        <td>enum</td>
        <td>
          Logs time window type<br/>
          <br/>
            <i>Enum</i>: 5m, 10m, 15m, 20m, 30m, 1h, 2h, 4h, 6h, 12h, 24h, 36h<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly.logsFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsanomaly)</sup></sup>



Filter to filter the logs with.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsanomalylogsfiltersimplefilter">simpleFilter</a></b></td>
        <td>object</td>
        <td>
          Simple lucene filter.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly.logsFilter.simpleFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsanomalylogsfilter)</sup></sup>



Simple lucene filter.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsanomalylogsfiltersimplefilterlabelfilters">labelFilters</a></b></td>
        <td>object</td>
        <td>
          Filter for labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>luceneQuery</b></td>
        <td>string</td>
        <td>
          The query.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly.logsFilter.simpleFilter.labelFilters
<sup><sup>[↩ Parent](#alertspecalerttypelogsanomalylogsfiltersimplefilter)</sup></sup>



Filter for labels.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsanomalylogsfiltersimplefilterlabelfiltersapplicationnameindex">applicationName</a></b></td>
        <td>[]object</td>
        <td>
          Application name to filter for.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>[]enum</td>
        <td>
          Severity to filter for.<br/>
          <br/>
            <i>Enum</i>: debug, info, warning, error, critical, verbose<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsanomalylogsfiltersimplefilterlabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          Subsystem name to filter for.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly.logsFilter.simpleFilter.labelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsanomalylogsfiltersimplefilterlabelfilters)</sup></sup>



Label filter specifications

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Operation to apply.<br/>
          <br/>
            <i>Enum</i>: is, includes, endsWith, startsWith<br/>
            <i>Default</i>: is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly.logsFilter.simpleFilter.labelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsanomalylogsfiltersimplefilterlabelfilters)</sup></sup>



Label filter specifications

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Operation to apply.<br/>
          <br/>
            <i>Enum</i>: is, includes, endsWith, startsWith<br/>
            <i>Default</i>: is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsImmediate
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>



Immediate alerts for logs.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsimmediatelogsfilter">logsFilter</a></b></td>
        <td>object</td>
        <td>
          Filter to filter the logs with.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>notificationPayloadFilter</b></td>
        <td>[]string</td>
        <td>
          Filter for the notification payload.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsImmediate.logsFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsimmediate)</sup></sup>



Filter to filter the logs with.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsimmediatelogsfiltersimplefilter">simpleFilter</a></b></td>
        <td>object</td>
        <td>
          Simple lucene filter.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsImmediate.logsFilter.simpleFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsimmediatelogsfilter)</sup></sup>



Simple lucene filter.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsimmediatelogsfiltersimplefilterlabelfilters">labelFilters</a></b></td>
        <td>object</td>
        <td>
          Filter for labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>luceneQuery</b></td>
        <td>string</td>
        <td>
          The query.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsImmediate.logsFilter.simpleFilter.labelFilters
<sup><sup>[↩ Parent](#alertspecalerttypelogsimmediatelogsfiltersimplefilter)</sup></sup>



Filter for labels.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsimmediatelogsfiltersimplefilterlabelfiltersapplicationnameindex">applicationName</a></b></td>
        <td>[]object</td>
        <td>
          Application name to filter for.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>[]enum</td>
        <td>
          Severity to filter for.<br/>
          <br/>
            <i>Enum</i>: debug, info, warning, error, critical, verbose<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsimmediatelogsfiltersimplefilterlabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          Subsystem name to filter for.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsImmediate.logsFilter.simpleFilter.labelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsimmediatelogsfiltersimplefilterlabelfilters)</sup></sup>



Label filter specifications

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Operation to apply.<br/>
          <br/>
            <i>Enum</i>: is, includes, endsWith, startsWith<br/>
            <i>Default</i>: is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsImmediate.logsFilter.simpleFilter.labelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsimmediatelogsfiltersimplefilterlabelfilters)</sup></sup>



Label filter specifications

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Operation to apply.<br/>
          <br/>
            <i>Enum</i>: is, includes, endsWith, startsWith<br/>
            <i>Default</i>: is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>



Alerts when a new log value appears.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsnewvaluelogsfilter">logsFilter</a></b></td>
        <td>object</td>
        <td>
          Filter to filter the logs with.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsnewvaluerulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          Rules that match the alert to the data.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>notificationPayloadFilter</b></td>
        <td>[]string</td>
        <td>
          Filter for the notification payload.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue.logsFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsnewvalue)</sup></sup>



Filter to filter the logs with.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsnewvaluelogsfiltersimplefilter">simpleFilter</a></b></td>
        <td>object</td>
        <td>
          Simple lucene filter.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue.logsFilter.simpleFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsnewvaluelogsfilter)</sup></sup>



Simple lucene filter.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsnewvaluelogsfiltersimplefilterlabelfilters">labelFilters</a></b></td>
        <td>object</td>
        <td>
          Filter for labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>luceneQuery</b></td>
        <td>string</td>
        <td>
          The query.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue.logsFilter.simpleFilter.labelFilters
<sup><sup>[↩ Parent](#alertspecalerttypelogsnewvaluelogsfiltersimplefilter)</sup></sup>



Filter for labels.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsnewvaluelogsfiltersimplefilterlabelfiltersapplicationnameindex">applicationName</a></b></td>
        <td>[]object</td>
        <td>
          Application name to filter for.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>[]enum</td>
        <td>
          Severity to filter for.<br/>
          <br/>
            <i>Enum</i>: debug, info, warning, error, critical, verbose<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsnewvaluelogsfiltersimplefilterlabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          Subsystem name to filter for.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue.logsFilter.simpleFilter.labelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsnewvaluelogsfiltersimplefilterlabelfilters)</sup></sup>



Label filter specifications

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Operation to apply.<br/>
          <br/>
            <i>Enum</i>: is, includes, endsWith, startsWith<br/>
            <i>Default</i>: is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue.logsFilter.simpleFilter.labelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsnewvaluelogsfiltersimplefilterlabelfilters)</sup></sup>



Label filter specifications

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Operation to apply.<br/>
          <br/>
            <i>Enum</i>: is, includes, endsWith, startsWith<br/>
            <i>Default</i>: is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsnewvalue)</sup></sup>



The rule to match the alert's conditions.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsnewvaluerulesindexcondition">condition</a></b></td>
        <td>object</td>
        <td>
          Condition to match to<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypelogsnewvaluerulesindex)</sup></sup>



Condition to match to

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>keypathToTrack</b></td>
        <td>string</td>
        <td>
          Where to look<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsnewvaluerulesindexconditiontimewindow">timeWindow</a></b></td>
        <td>object</td>
        <td>
          Which time window.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue.rules[index].condition.timeWindow
<sup><sup>[↩ Parent](#alertspecalerttypelogsnewvaluerulesindexcondition)</sup></sup>



Which time window.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>specificValue</b></td>
        <td>enum</td>
        <td>
          Time windows.<br/>
          <br/>
            <i>Enum</i>: 12h, 24h, 48h, 72h, 1w, 1mo, 2mo, 3mo<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>



Alerts for when a log exceeds a defined ratio.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholddenominator">denominator</a></b></td>
        <td>object</td>
        <td>
          A filter for logs.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>denominatorAlias</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholdnumerator">numerator</a></b></td>
        <td>object</td>
        <td>
          A filter for logs.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>numeratorAlias</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholdrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          Rules that match the alert to the data.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.denominator
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothreshold)</sup></sup>



A filter for logs.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholddenominatorsimplefilter">simpleFilter</a></b></td>
        <td>object</td>
        <td>
          Simple lucene filter.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.denominator.simpleFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholddenominator)</sup></sup>



Simple lucene filter.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholddenominatorsimplefilterlabelfilters">labelFilters</a></b></td>
        <td>object</td>
        <td>
          Filter for labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>luceneQuery</b></td>
        <td>string</td>
        <td>
          The query.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.denominator.simpleFilter.labelFilters
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholddenominatorsimplefilter)</sup></sup>



Filter for labels.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholddenominatorsimplefilterlabelfiltersapplicationnameindex">applicationName</a></b></td>
        <td>[]object</td>
        <td>
          Application name to filter for.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>[]enum</td>
        <td>
          Severity to filter for.<br/>
          <br/>
            <i>Enum</i>: debug, info, warning, error, critical, verbose<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholddenominatorsimplefilterlabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          Subsystem name to filter for.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.denominator.simpleFilter.labelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholddenominatorsimplefilterlabelfilters)</sup></sup>



Label filter specifications

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Operation to apply.<br/>
          <br/>
            <i>Enum</i>: is, includes, endsWith, startsWith<br/>
            <i>Default</i>: is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.denominator.simpleFilter.labelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholddenominatorsimplefilterlabelfilters)</sup></sup>



Label filter specifications

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Operation to apply.<br/>
          <br/>
            <i>Enum</i>: is, includes, endsWith, startsWith<br/>
            <i>Default</i>: is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.numerator
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothreshold)</sup></sup>



A filter for logs.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholdnumeratorsimplefilter">simpleFilter</a></b></td>
        <td>object</td>
        <td>
          Simple lucene filter.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.numerator.simpleFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholdnumerator)</sup></sup>



Simple lucene filter.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholdnumeratorsimplefilterlabelfilters">labelFilters</a></b></td>
        <td>object</td>
        <td>
          Filter for labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>luceneQuery</b></td>
        <td>string</td>
        <td>
          The query.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.numerator.simpleFilter.labelFilters
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholdnumeratorsimplefilter)</sup></sup>



Filter for labels.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholdnumeratorsimplefilterlabelfiltersapplicationnameindex">applicationName</a></b></td>
        <td>[]object</td>
        <td>
          Application name to filter for.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>[]enum</td>
        <td>
          Severity to filter for.<br/>
          <br/>
            <i>Enum</i>: debug, info, warning, error, critical, verbose<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholdnumeratorsimplefilterlabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          Subsystem name to filter for.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.numerator.simpleFilter.labelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholdnumeratorsimplefilterlabelfilters)</sup></sup>



Label filter specifications

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Operation to apply.<br/>
          <br/>
            <i>Enum</i>: is, includes, endsWith, startsWith<br/>
            <i>Default</i>: is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.numerator.simpleFilter.labelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholdnumeratorsimplefilterlabelfilters)</sup></sup>



Label filter specifications

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Operation to apply.<br/>
          <br/>
            <i>Enum</i>: is, includes, endsWith, startsWith<br/>
            <i>Default</i>: is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothreshold)</sup></sup>



The rule to match the alert's conditions.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholdrulesindexcondition">condition</a></b></td>
        <td>object</td>
        <td>
          Condition to match<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholdrulesindexoverride">override</a></b></td>
        <td>object</td>
        <td>
          Override alert properties<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholdrulesindex)</sup></sup>



Condition to match

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>conditionType</b></td>
        <td>enum</td>
        <td>
          Condition to evaluate with.<br/>
          <br/>
            <i>Enum</i>: moreThan, lessThan<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>threshold</b></td>
        <td>int or string</td>
        <td>
          Threshold to pass.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholdrulesindexconditiontimewindow">timeWindow</a></b></td>
        <td>object</td>
        <td>
          Time window to evaluate.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.rules[index].condition.timeWindow
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholdrulesindexcondition)</sup></sup>



Time window to evaluate.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>specificValue</b></td>
        <td>enum</td>
        <td>
          Time window type.<br/>
          <br/>
            <i>Enum</i>: 5m, 10m, 15m, 30m, 1h, 2h, 4h, 6h, 12h, 24h, 36h<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.rules[index].override
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholdrulesindex)</sup></sup>



Override alert properties

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>priority</b></td>
        <td>enum</td>
        <td>
          Priority to override it<br/>
          <br/>
            <i>Enum</i>: p1, p2, p3, p4, p5<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>



Alerts for when a log crosses a threshold.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsthresholdrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          Rules that match the alert to the data.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsthresholdlogsfilter">logsFilter</a></b></td>
        <td>object</td>
        <td>
          Filter to filter the logs with.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>notificationPayloadFilter</b></td>
        <td>[]string</td>
        <td>
          Filter for the notification payload.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsthresholdundetectedvaluesmanagement">undetectedValuesManagement</a></b></td>
        <td>object</td>
        <td>
          How to work with undetected values.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsthreshold)</sup></sup>



The rule to match the alert's conditions.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsthresholdrulesindexcondition">condition</a></b></td>
        <td>object</td>
        <td>
          Condition to match<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsthresholdrulesindexoverride">override</a></b></td>
        <td>object</td>
        <td>
          Alert overrides.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypelogsthresholdrulesindex)</sup></sup>



Condition to match

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>logsThresholdConditionType</b></td>
        <td>enum</td>
        <td>
          Condition type.<br/>
          <br/>
            <i>Enum</i>: moreThan, lessThan<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>threshold</b></td>
        <td>int or string</td>
        <td>
          Threshold to match to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsthresholdrulesindexconditiontimewindow">timeWindow</a></b></td>
        <td>object</td>
        <td>
          Time window in which the condition is checked.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.rules[index].condition.timeWindow
<sup><sup>[↩ Parent](#alertspecalerttypelogsthresholdrulesindexcondition)</sup></sup>



Time window in which the condition is checked.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>specificValue</b></td>
        <td>enum</td>
        <td>
          Logs time window type<br/>
          <br/>
            <i>Enum</i>: 5m, 10m, 15m, 20m, 30m, 1h, 2h, 4h, 6h, 12h, 24h, 36h<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.rules[index].override
<sup><sup>[↩ Parent](#alertspecalerttypelogsthresholdrulesindex)</sup></sup>



Alert overrides.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>priority</b></td>
        <td>enum</td>
        <td>
          Priority to override it<br/>
          <br/>
            <i>Enum</i>: p1, p2, p3, p4, p5<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.logsFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsthreshold)</sup></sup>



Filter to filter the logs with.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsthresholdlogsfiltersimplefilter">simpleFilter</a></b></td>
        <td>object</td>
        <td>
          Simple lucene filter.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.logsFilter.simpleFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsthresholdlogsfilter)</sup></sup>



Simple lucene filter.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsthresholdlogsfiltersimplefilterlabelfilters">labelFilters</a></b></td>
        <td>object</td>
        <td>
          Filter for labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>luceneQuery</b></td>
        <td>string</td>
        <td>
          The query.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.logsFilter.simpleFilter.labelFilters
<sup><sup>[↩ Parent](#alertspecalerttypelogsthresholdlogsfiltersimplefilter)</sup></sup>



Filter for labels.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsthresholdlogsfiltersimplefilterlabelfiltersapplicationnameindex">applicationName</a></b></td>
        <td>[]object</td>
        <td>
          Application name to filter for.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>[]enum</td>
        <td>
          Severity to filter for.<br/>
          <br/>
            <i>Enum</i>: debug, info, warning, error, critical, verbose<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsthresholdlogsfiltersimplefilterlabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          Subsystem name to filter for.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.logsFilter.simpleFilter.labelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsthresholdlogsfiltersimplefilterlabelfilters)</sup></sup>



Label filter specifications

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Operation to apply.<br/>
          <br/>
            <i>Enum</i>: is, includes, endsWith, startsWith<br/>
            <i>Default</i>: is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.logsFilter.simpleFilter.labelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsthresholdlogsfiltersimplefilterlabelfilters)</sup></sup>



Label filter specifications

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Operation to apply.<br/>
          <br/>
            <i>Enum</i>: is, includes, endsWith, startsWith<br/>
            <i>Default</i>: is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.undetectedValuesManagement
<sup><sup>[↩ Parent](#alertspecalerttypelogsthreshold)</sup></sup>



How to work with undetected values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>autoRetireTimeframe</b></td>
        <td>enum</td>
        <td>
          Automatically retire the alerts after this time.<br/>
          <br/>
            <i>Enum</i>: never, 5m, 10m, 1h, 2h, 6h, 12h, 24h<br/>
            <i>Default</i>: never<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>triggerUndetectedValues</b></td>
        <td>boolean</td>
        <td>
          Deactivate triggering the alert on undetected values.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>



Alerts are sent when the number of logs matching a filter is more than or less than a threshold over a specific time window.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>ignoreInfinity</b></td>
        <td>boolean</td>
        <td>
          Ignore infinity on the threshold value.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogstimerelativethresholdlogsfilter">logsFilter</a></b></td>
        <td>object</td>
        <td>
          A filter for logs.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogstimerelativethresholdrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          Rules that match the alert to the data.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>notificationPayloadFilter</b></td>
        <td>[]string</td>
        <td>
          Filter for the notification payload.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogstimerelativethresholdundetectedvaluesmanagement">undetectedValuesManagement</a></b></td>
        <td>object</td>
        <td>
          How to work with undetected values.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.logsFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethreshold)</sup></sup>



A filter for logs.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogstimerelativethresholdlogsfiltersimplefilter">simpleFilter</a></b></td>
        <td>object</td>
        <td>
          Simple lucene filter.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.logsFilter.simpleFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethresholdlogsfilter)</sup></sup>



Simple lucene filter.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogstimerelativethresholdlogsfiltersimplefilterlabelfilters">labelFilters</a></b></td>
        <td>object</td>
        <td>
          Filter for labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>luceneQuery</b></td>
        <td>string</td>
        <td>
          The query.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.logsFilter.simpleFilter.labelFilters
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethresholdlogsfiltersimplefilter)</sup></sup>



Filter for labels.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogstimerelativethresholdlogsfiltersimplefilterlabelfiltersapplicationnameindex">applicationName</a></b></td>
        <td>[]object</td>
        <td>
          Application name to filter for.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>[]enum</td>
        <td>
          Severity to filter for.<br/>
          <br/>
            <i>Enum</i>: debug, info, warning, error, critical, verbose<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogstimerelativethresholdlogsfiltersimplefilterlabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          Subsystem name to filter for.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.logsFilter.simpleFilter.labelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethresholdlogsfiltersimplefilterlabelfilters)</sup></sup>



Label filter specifications

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Operation to apply.<br/>
          <br/>
            <i>Enum</i>: is, includes, endsWith, startsWith<br/>
            <i>Default</i>: is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.logsFilter.simpleFilter.labelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethresholdlogsfiltersimplefilterlabelfilters)</sup></sup>



Label filter specifications

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Operation to apply.<br/>
          <br/>
            <i>Enum</i>: is, includes, endsWith, startsWith<br/>
            <i>Default</i>: is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethreshold)</sup></sup>



The rule to match the alert's conditions.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogstimerelativethresholdrulesindexcondition">condition</a></b></td>
        <td>object</td>
        <td>
          The condition to match to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogstimerelativethresholdrulesindexoverride">override</a></b></td>
        <td>object</td>
        <td>
          Override alert properties<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethresholdrulesindex)</sup></sup>



The condition to match to.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>comparedTo</b></td>
        <td>enum</td>
        <td>
          Comparison window.<br/>
          <br/>
            <i>Enum</i>: previousHour, sameHourYesterday, sameHourLastWeek, yesterday, sameDayLastWeek, sameDayLastMonth<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>conditionType</b></td>
        <td>enum</td>
        <td>
          How to compare.<br/>
          <br/>
            <i>Enum</i>: moreThan, lessThan<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>threshold</b></td>
        <td>int or string</td>
        <td>
          Threshold to match.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.rules[index].override
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethresholdrulesindex)</sup></sup>



Override alert properties

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>priority</b></td>
        <td>enum</td>
        <td>
          Priority to override it<br/>
          <br/>
            <i>Enum</i>: p1, p2, p3, p4, p5<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.undetectedValuesManagement
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethreshold)</sup></sup>



How to work with undetected values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>autoRetireTimeframe</b></td>
        <td>enum</td>
        <td>
          Automatically retire the alerts after this time.<br/>
          <br/>
            <i>Enum</i>: never, 5m, 10m, 1h, 2h, 6h, 12h, 24h<br/>
            <i>Default</i>: never<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>triggerUndetectedValues</b></td>
        <td>boolean</td>
        <td>
          Deactivate triggering the alert on undetected values.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>



Alerts for unique count changes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsuniquecountlogsfilter">logsFilter</a></b></td>
        <td>object</td>
        <td>
          Filter to filter the logs with.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsuniquecountrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          Rules that match the alert to the data.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>uniqueCountKeypath</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>maxUniqueCountPerGroupByKey</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>notificationPayloadFilter</b></td>
        <td>[]string</td>
        <td>
          Filter for the notification payload.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount.logsFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsuniquecount)</sup></sup>



Filter to filter the logs with.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsuniquecountlogsfiltersimplefilter">simpleFilter</a></b></td>
        <td>object</td>
        <td>
          Simple lucene filter.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount.logsFilter.simpleFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsuniquecountlogsfilter)</sup></sup>



Simple lucene filter.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsuniquecountlogsfiltersimplefilterlabelfilters">labelFilters</a></b></td>
        <td>object</td>
        <td>
          Filter for labels.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>luceneQuery</b></td>
        <td>string</td>
        <td>
          The query.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount.logsFilter.simpleFilter.labelFilters
<sup><sup>[↩ Parent](#alertspecalerttypelogsuniquecountlogsfiltersimplefilter)</sup></sup>



Filter for labels.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsuniquecountlogsfiltersimplefilterlabelfiltersapplicationnameindex">applicationName</a></b></td>
        <td>[]object</td>
        <td>
          Application name to filter for.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>[]enum</td>
        <td>
          Severity to filter for.<br/>
          <br/>
            <i>Enum</i>: debug, info, warning, error, critical, verbose<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsuniquecountlogsfiltersimplefilterlabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          Subsystem name to filter for.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount.logsFilter.simpleFilter.labelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsuniquecountlogsfiltersimplefilterlabelfilters)</sup></sup>



Label filter specifications

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Operation to apply.<br/>
          <br/>
            <i>Enum</i>: is, includes, endsWith, startsWith<br/>
            <i>Default</i>: is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount.logsFilter.simpleFilter.labelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsuniquecountlogsfiltersimplefilterlabelfilters)</sup></sup>



Label filter specifications

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Operation to apply.<br/>
          <br/>
            <i>Enum</i>: is, includes, endsWith, startsWith<br/>
            <i>Default</i>: is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          The value<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsuniquecount)</sup></sup>



The rule to match the alert's conditions.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypelogsuniquecountrulesindexcondition">condition</a></b></td>
        <td>object</td>
        <td>
          Condition to match to.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypelogsuniquecountrulesindex)</sup></sup>



Condition to match to.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>threshold</b></td>
        <td>integer</td>
        <td>
          Threshold to cross<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsuniquecountrulesindexconditiontimewindow">timeWindow</a></b></td>
        <td>object</td>
        <td>
          Time window to evaluate.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount.rules[index].condition.timeWindow
<sup><sup>[↩ Parent](#alertspecalerttypelogsuniquecountrulesindexcondition)</sup></sup>



Time window to evaluate.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>specificValue</b></td>
        <td>enum</td>
        <td>
          Time windows for Logs Unique Count<br/>
          <br/>
            <i>Enum</i>: 1m, 5m, 10m, 15m, 20m, 30m, 1h, 2h, 4h, 6h, 12h, 24h, 36h<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricAnomaly
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>



Anomaly alerts for metrics.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypemetricanomalymetricfilter">metricFilter</a></b></td>
        <td>object</td>
        <td>
          PromQL filter for metrics<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricanomalyrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          Rules that match the alert to the data.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricAnomaly.metricFilter
<sup><sup>[↩ Parent](#alertspecalerttypemetricanomaly)</sup></sup>



PromQL filter for metrics

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>promql</b></td>
        <td>string</td>
        <td>
          PromQL query: https://coralogix.com/academy/mastering-metrics-in-coralogix/promql-fundamentals/<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricAnomaly.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypemetricanomaly)</sup></sup>



The rule to match the alert's conditions.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypemetricanomalyrulesindexcondition">condition</a></b></td>
        <td>object</td>
        <td>
          Condition to match to.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricAnomaly.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypemetricanomalyrulesindex)</sup></sup>



Condition to match to.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>conditionType</b></td>
        <td>enum</td>
        <td>
          Condition type.<br/>
          <br/>
            <i>Enum</i>: moreThanUsual, lessThanUsual<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>forOverPct</b></td>
        <td>integer</td>
        <td>
          Percentage for the threshold<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Maximum</i>: 100<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>minNonNullValuesPct</b></td>
        <td>integer</td>
        <td>
          Replace with a number<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Maximum</i>: 100<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricanomalyrulesindexconditionofthelast">ofTheLast</a></b></td>
        <td>object</td>
        <td>
          Time window to match within<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>threshold</b></td>
        <td>int or string</td>
        <td>
          Threshold to clear.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricAnomaly.rules[index].condition.ofTheLast
<sup><sup>[↩ Parent](#alertspecalerttypemetricanomalyrulesindexcondition)</sup></sup>



Time window to match within

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>specificValue</b></td>
        <td>enum</td>
        <td>
          Time window type.<br/>
          <br/>
            <i>Enum</i>: 1m, 5m, 10m, 15m, 20m, 30m, 1h, 2h, 4h, 6h, 12h, 24h, 36h<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricThreshold
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>



Alerts for when a metric crosses a threshold.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypemetricthresholdmetricfilter">metricFilter</a></b></td>
        <td>object</td>
        <td>
          Filter for metrics<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricthresholdmissingvalues">missingValues</a></b></td>
        <td>object</td>
        <td>
          Missing values strategies.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricthresholdrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          Rules that match the alert to the data.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricthresholdundetectedvaluesmanagement">undetectedValuesManagement</a></b></td>
        <td>object</td>
        <td>
          How to work with undetected values.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricThreshold.metricFilter
<sup><sup>[↩ Parent](#alertspecalerttypemetricthreshold)</sup></sup>



Filter for metrics

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>promql</b></td>
        <td>string</td>
        <td>
          PromQL query: https://coralogix.com/academy/mastering-metrics-in-coralogix/promql-fundamentals/<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricThreshold.missingValues
<sup><sup>[↩ Parent](#alertspecalerttypemetricthreshold)</sup></sup>



Missing values strategies.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>minNonNullValuesPct</b></td>
        <td>integer</td>
        <td>
          Replace with a number<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Maximum</i>: 100<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replaceWithZero</b></td>
        <td>boolean</td>
        <td>
          Replace missing values with 0s<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricThreshold.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypemetricthreshold)</sup></sup>



Rules that match the alert to the data.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypemetricthresholdrulesindexcondition">condition</a></b></td>
        <td>object</td>
        <td>
          Conditions to match for the rule.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricthresholdrulesindexoverride">override</a></b></td>
        <td>object</td>
        <td>
          Alert property overrides<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricThreshold.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypemetricthresholdrulesindex)</sup></sup>



Conditions to match for the rule.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>conditionType</b></td>
        <td>enum</td>
        <td>
          ConditionType type.<br/>
          <br/>
            <i>Enum</i>: moreThan, lessThan, moreThanOrEquals, lessThanOrEquals<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>forOverPct</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Maximum</i>: 100<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricthresholdrulesindexconditionofthelast">ofTheLast</a></b></td>
        <td>object</td>
        <td>
          Time window type.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.specificValue) != has(self.dynamicDuration): Exactly one of specificValue or dynamicDuration is required</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>threshold</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricThreshold.rules[index].condition.ofTheLast
<sup><sup>[↩ Parent](#alertspecalerttypemetricthresholdrulesindexcondition)</sup></sup>



Time window type.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>dynamicDuration</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>specificValue</b></td>
        <td>enum</td>
        <td>
          Time window type.<br/>
          <br/>
            <i>Enum</i>: 1m, 5m, 10m, 15m, 20m, 30m, 1h, 2h, 4h, 6h, 12h, 24h, 36h<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricThreshold.rules[index].override
<sup><sup>[↩ Parent](#alertspecalerttypemetricthresholdrulesindex)</sup></sup>



Alert property overrides

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>priority</b></td>
        <td>enum</td>
        <td>
          Priority to override it<br/>
          <br/>
            <i>Enum</i>: p1, p2, p3, p4, p5<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricThreshold.undetectedValuesManagement
<sup><sup>[↩ Parent](#alertspecalerttypemetricthreshold)</sup></sup>



How to work with undetected values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>autoRetireTimeframe</b></td>
        <td>enum</td>
        <td>
          Automatically retire the alerts after this time.<br/>
          <br/>
            <i>Enum</i>: never, 5m, 10m, 1h, 2h, 6h, 12h, 24h<br/>
            <i>Default</i>: never<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>triggerUndetectedValues</b></td>
        <td>boolean</td>
        <td>
          Deactivate triggering the alert on undetected values.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingImmediate
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>



Immediate alerts for traces.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>notificationPayloadFilter</b></td>
        <td>[]string</td>
        <td>
          Filter for the notification payload.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingimmediatetracingfilter">tracingFilter</a></b></td>
        <td>object</td>
        <td>
          A simple tracing filter.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingImmediate.tracingFilter
<sup><sup>[↩ Parent](#alertspecalerttypetracingimmediate)</sup></sup>



A simple tracing filter.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypetracingimmediatetracingfiltersimple">simple</a></b></td>
        <td>object</td>
        <td>
          Simple tracing filter paired with a latency.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingImmediate.tracingFilter.simple
<sup><sup>[↩ Parent](#alertspecalerttypetracingimmediatetracingfilter)</sup></sup>



Simple tracing filter paired with a latency.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>latencyThresholdMs</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingimmediatetracingfiltersimpletracinglabelfilters">tracingLabelFilters</a></b></td>
        <td>object</td>
        <td>
          Filter for traces.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingImmediate.tracingFilter.simple.tracingLabelFilters
<sup><sup>[↩ Parent](#alertspecalerttypetracingimmediatetracingfiltersimple)</sup></sup>



Filter for traces.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypetracingimmediatetracingfiltersimpletracinglabelfiltersapplicationnameindex">applicationName</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingimmediatetracingfiltersimpletracinglabelfiltersoperationnameindex">operationName</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingimmediatetracingfiltersimpletracinglabelfiltersservicenameindex">serviceName</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingimmediatetracingfiltersimpletracinglabelfiltersspanfieldsindex">spanFields</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingimmediatetracingfiltersimpletracinglabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingImmediate.tracingFilter.simple.tracingLabelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypetracingimmediatetracingfiltersimpletracinglabelfilters)</sup></sup>



Filter - values and operation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Tracing filter operations.<br/>
          <br/>
            <i>Enum</i>: includes, endsWith, startsWith, isNot, is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingImmediate.tracingFilter.simple.tracingLabelFilters.operationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypetracingimmediatetracingfiltersimpletracinglabelfilters)</sup></sup>



Filter - values and operation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Tracing filter operations.<br/>
          <br/>
            <i>Enum</i>: includes, endsWith, startsWith, isNot, is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingImmediate.tracingFilter.simple.tracingLabelFilters.serviceName[index]
<sup><sup>[↩ Parent](#alertspecalerttypetracingimmediatetracingfiltersimpletracinglabelfilters)</sup></sup>



Filter - values and operation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Tracing filter operations.<br/>
          <br/>
            <i>Enum</i>: includes, endsWith, startsWith, isNot, is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingImmediate.tracingFilter.simple.tracingLabelFilters.spanFields[index]
<sup><sup>[↩ Parent](#alertspecalerttypetracingimmediatetracingfiltersimpletracinglabelfilters)</sup></sup>



Filter for spans

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypetracingimmediatetracingfiltersimpletracinglabelfiltersspanfieldsindexfiltertype">filterType</a></b></td>
        <td>object</td>
        <td>
          Filter - values and operation.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingImmediate.tracingFilter.simple.tracingLabelFilters.spanFields[index].filterType
<sup><sup>[↩ Parent](#alertspecalerttypetracingimmediatetracingfiltersimpletracinglabelfiltersspanfieldsindex)</sup></sup>



Filter - values and operation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Tracing filter operations.<br/>
          <br/>
            <i>Enum</i>: includes, endsWith, startsWith, isNot, is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingImmediate.tracingFilter.simple.tracingLabelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypetracingimmediatetracingfiltersimpletracinglabelfilters)</sup></sup>



Filter - values and operation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Tracing filter operations.<br/>
          <br/>
            <i>Enum</i>: includes, endsWith, startsWith, isNot, is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>



Alerts for when traces crosses a threshold.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypetracingthresholdrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          Rules that match the alert to the data.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>notificationPayloadFilter</b></td>
        <td>[]string</td>
        <td>
          Filter for the notification payload.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingthresholdtracingfilter">tracingFilter</a></b></td>
        <td>object</td>
        <td>
          Filter the base collection.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypetracingthreshold)</sup></sup>



The rule to match the alert's conditions.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypetracingthresholdrulesindexcondition">condition</a></b></td>
        <td>object</td>
        <td>
          The condition to match to.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypetracingthresholdrulesindex)</sup></sup>



The condition to match to.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>spanAmount</b></td>
        <td>int or string</td>
        <td>
          Threshold amount.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingthresholdrulesindexconditiontimewindow">timeWindow</a></b></td>
        <td>object</td>
        <td>
          Time window to evaluate.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.rules[index].condition.timeWindow
<sup><sup>[↩ Parent](#alertspecalerttypetracingthresholdrulesindexcondition)</sup></sup>



Time window to evaluate.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>specificValue</b></td>
        <td>enum</td>
        <td>
          Time window type for tracing.<br/>
          <br/>
            <i>Enum</i>: 5m, 10m, 15m, 20m, 30m, 1h, 2h, 4h, 6h, 12h, 24h, 36h<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.tracingFilter
<sup><sup>[↩ Parent](#alertspecalerttypetracingthreshold)</sup></sup>



Filter the base collection.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypetracingthresholdtracingfiltersimple">simple</a></b></td>
        <td>object</td>
        <td>
          Simple tracing filter paired with a latency.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.tracingFilter.simple
<sup><sup>[↩ Parent](#alertspecalerttypetracingthresholdtracingfilter)</sup></sup>



Simple tracing filter paired with a latency.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>latencyThresholdMs</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingthresholdtracingfiltersimpletracinglabelfilters">tracingLabelFilters</a></b></td>
        <td>object</td>
        <td>
          Filter for traces.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.tracingFilter.simple.tracingLabelFilters
<sup><sup>[↩ Parent](#alertspecalerttypetracingthresholdtracingfiltersimple)</sup></sup>



Filter for traces.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypetracingthresholdtracingfiltersimpletracinglabelfiltersapplicationnameindex">applicationName</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingthresholdtracingfiltersimpletracinglabelfiltersoperationnameindex">operationName</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingthresholdtracingfiltersimpletracinglabelfiltersservicenameindex">serviceName</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingthresholdtracingfiltersimpletracinglabelfiltersspanfieldsindex">spanFields</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingthresholdtracingfiltersimpletracinglabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.tracingFilter.simple.tracingLabelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypetracingthresholdtracingfiltersimpletracinglabelfilters)</sup></sup>



Filter - values and operation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Tracing filter operations.<br/>
          <br/>
            <i>Enum</i>: includes, endsWith, startsWith, isNot, is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.tracingFilter.simple.tracingLabelFilters.operationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypetracingthresholdtracingfiltersimpletracinglabelfilters)</sup></sup>



Filter - values and operation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Tracing filter operations.<br/>
          <br/>
            <i>Enum</i>: includes, endsWith, startsWith, isNot, is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.tracingFilter.simple.tracingLabelFilters.serviceName[index]
<sup><sup>[↩ Parent](#alertspecalerttypetracingthresholdtracingfiltersimpletracinglabelfilters)</sup></sup>



Filter - values and operation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Tracing filter operations.<br/>
          <br/>
            <i>Enum</i>: includes, endsWith, startsWith, isNot, is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.tracingFilter.simple.tracingLabelFilters.spanFields[index]
<sup><sup>[↩ Parent](#alertspecalerttypetracingthresholdtracingfiltersimpletracinglabelfilters)</sup></sup>



Filter for spans

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypetracingthresholdtracingfiltersimpletracinglabelfiltersspanfieldsindexfiltertype">filterType</a></b></td>
        <td>object</td>
        <td>
          Filter - values and operation.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.tracingFilter.simple.tracingLabelFilters.spanFields[index].filterType
<sup><sup>[↩ Parent](#alertspecalerttypetracingthresholdtracingfiltersimpletracinglabelfiltersspanfieldsindex)</sup></sup>



Filter - values and operation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Tracing filter operations.<br/>
          <br/>
            <i>Enum</i>: includes, endsWith, startsWith, isNot, is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.tracingFilter.simple.tracingLabelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypetracingthresholdtracingfiltersimpletracinglabelfilters)</sup></sup>



Filter - values and operation.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          Tracing filter operations.<br/>
          <br/>
            <i>Enum</i>: includes, endsWith, startsWith, isNot, is<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.incidentsSettings
<sup><sup>[↩ Parent](#alertspec)</sup></sup>



Settings for the attached incidents.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>notifyOn</b></td>
        <td>enum</td>
        <td>
          When to notify.<br/>
          <br/>
            <i>Enum</i>: triggeredOnly, triggeredAndResolved<br/>
            <i>Default</i>: triggeredOnly<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecincidentssettingsretriggeringperiod">retriggeringPeriod</a></b></td>
        <td>object</td>
        <td>
          When to re-notify.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.incidentsSettings.retriggeringPeriod
<sup><sup>[↩ Parent](#alertspecincidentssettings)</sup></sup>



When to re-notify.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>minutes</b></td>
        <td>integer</td>
        <td>
          Delay between re-triggered alerts.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup
<sup><sup>[↩ Parent](#alertspec)</sup></sup>



Where notifications should be sent to.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupdestinationsindex">destinations</a></b></td>
        <td>[]object</td>
        <td>
          The destinations for notifications (Notification Center feature).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>groupByKeys</b></td>
        <td>[]string</td>
        <td>
          Group notification by these keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgrouprouter">router</a></b></td>
        <td>object</td>
        <td>
          The router for notifications (Notification Center feature).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupwebhooksindex">webhooks</a></b></td>
        <td>[]object</td>
        <td>
          Webhooks to trigger for notifications.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.destinations[index]
<sup><sup>[↩ Parent](#alertspecnotificationgroup)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupdestinationsindexconnector">connector</a></b></td>
        <td>object</td>
        <td>
          Connector is the connector for the destination. Should be one of backendRef or resourceRef.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.backendRef) != has(self.resourceRef): Exactly one of backendRef or resourceRef must be set</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>notifyOn</b></td>
        <td>enum</td>
        <td>
          When to notify.<br/>
          <br/>
            <i>Enum</i>: triggeredOnly, triggeredAndResolved<br/>
            <i>Default</i>: triggeredOnly<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupdestinationsindextriggeredroutingoverrides">triggeredRoutingOverrides</a></b></td>
        <td>object</td>
        <td>
          The routing configuration to override from the connector/preset for triggered notifications.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupdestinationsindexpreset">preset</a></b></td>
        <td>object</td>
        <td>
          Preset is the preset for the destination. Should be one of backendRef or resourceRef.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.backendRef) != has(self.resourceRef): Exactly one of backendRef or resourceRef must be set</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupdestinationsindexresolvedroutingoverrides">resolvedRoutingOverrides</a></b></td>
        <td>object</td>
        <td>
          Optional routing configuration to override from the connector/preset for resolved notifications.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.destinations[index].connector
<sup><sup>[↩ Parent](#alertspecnotificationgroupdestinationsindex)</sup></sup>



Connector is the connector for the destination. Should be one of backendRef or resourceRef.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupdestinationsindexconnectorbackendref">backendRef</a></b></td>
        <td>object</td>
        <td>
          BackendRef is a reference to a backend resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupdestinationsindexconnectorresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          ResourceRef is a reference to a Kubernetes resource.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.destinations[index].connector.backendRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupdestinationsindexconnector)</sup></sup>



BackendRef is a reference to a backend resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.destinations[index].connector.resourceRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupdestinationsindexconnector)</sup></sup>



ResourceRef is a reference to a Kubernetes resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.destinations[index].triggeredRoutingOverrides
<sup><sup>[↩ Parent](#alertspecnotificationgroupdestinationsindex)</sup></sup>



The routing configuration to override from the connector/preset for triggered notifications.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupdestinationsindextriggeredroutingoverridesconfigoverrides">configOverrides</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.destinations[index].triggeredRoutingOverrides.configOverrides
<sup><sup>[↩ Parent](#alertspecnotificationgroupdestinationsindextriggeredroutingoverrides)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>outputSchemaId</b></td>
        <td>string</td>
        <td>
          The ID of the output schema to use for routing notifications<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupdestinationsindextriggeredroutingoverridesconfigoverridesconnectorconfigfieldsindex">connectorConfigFields</a></b></td>
        <td>[]object</td>
        <td>
          Connector configuration fields.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupdestinationsindextriggeredroutingoverridesconfigoverridesmessageconfigfieldsindex">messageConfigFields</a></b></td>
        <td>[]object</td>
        <td>
          Notification message configuration fields.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.destinations[index].triggeredRoutingOverrides.configOverrides.connectorConfigFields[index]
<sup><sup>[↩ Parent](#alertspecnotificationgroupdestinationsindextriggeredroutingoverridesconfigoverrides)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldName</b></td>
        <td>string</td>
        <td>
          The name of the configuration field.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>template</b></td>
        <td>string</td>
        <td>
          The template for the configuration field.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.destinations[index].triggeredRoutingOverrides.configOverrides.messageConfigFields[index]
<sup><sup>[↩ Parent](#alertspecnotificationgroupdestinationsindextriggeredroutingoverridesconfigoverrides)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldName</b></td>
        <td>string</td>
        <td>
          The name of the configuration field.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>template</b></td>
        <td>string</td>
        <td>
          The template for the configuration field.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.destinations[index].preset
<sup><sup>[↩ Parent](#alertspecnotificationgroupdestinationsindex)</sup></sup>



Preset is the preset for the destination. Should be one of backendRef or resourceRef.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupdestinationsindexpresetbackendref">backendRef</a></b></td>
        <td>object</td>
        <td>
          BackendRef is a reference to a backend resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupdestinationsindexpresetresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          ResourceRef is a reference to a Kubernetes resource.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.destinations[index].preset.backendRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupdestinationsindexpreset)</sup></sup>



BackendRef is a reference to a backend resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.destinations[index].preset.resourceRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupdestinationsindexpreset)</sup></sup>



ResourceRef is a reference to a Kubernetes resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.destinations[index].resolvedRoutingOverrides
<sup><sup>[↩ Parent](#alertspecnotificationgroupdestinationsindex)</sup></sup>



Optional routing configuration to override from the connector/preset for resolved notifications.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupdestinationsindexresolvedroutingoverridesconfigoverrides">configOverrides</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.destinations[index].resolvedRoutingOverrides.configOverrides
<sup><sup>[↩ Parent](#alertspecnotificationgroupdestinationsindexresolvedroutingoverrides)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>outputSchemaId</b></td>
        <td>string</td>
        <td>
          The ID of the output schema to use for routing notifications<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupdestinationsindexresolvedroutingoverridesconfigoverridesconnectorconfigfieldsindex">connectorConfigFields</a></b></td>
        <td>[]object</td>
        <td>
          Connector configuration fields.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupdestinationsindexresolvedroutingoverridesconfigoverridesmessageconfigfieldsindex">messageConfigFields</a></b></td>
        <td>[]object</td>
        <td>
          Notification message configuration fields.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.destinations[index].resolvedRoutingOverrides.configOverrides.connectorConfigFields[index]
<sup><sup>[↩ Parent](#alertspecnotificationgroupdestinationsindexresolvedroutingoverridesconfigoverrides)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldName</b></td>
        <td>string</td>
        <td>
          The name of the configuration field.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>template</b></td>
        <td>string</td>
        <td>
          The template for the configuration field.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.destinations[index].resolvedRoutingOverrides.configOverrides.messageConfigFields[index]
<sup><sup>[↩ Parent](#alertspecnotificationgroupdestinationsindexresolvedroutingoverridesconfigoverrides)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldName</b></td>
        <td>string</td>
        <td>
          The name of the configuration field.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>template</b></td>
        <td>string</td>
        <td>
          The template for the configuration field.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.router
<sup><sup>[↩ Parent](#alertspecnotificationgroup)</sup></sup>



The router for notifications (Notification Center feature).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>notifyOn</b></td>
        <td>enum</td>
        <td>
          When to notify.<br/>
          <br/>
            <i>Enum</i>: triggeredOnly, triggeredAndResolved<br/>
            <i>Default</i>: triggeredOnly<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.webhooks[index]
<sup><sup>[↩ Parent](#alertspecnotificationgroup)</sup></sup>



Settings for a notification webhook.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupwebhooksindexintegration">integration</a></b></td>
        <td>object</td>
        <td>
          Type and spec of webhook.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.integrationRef) || has(self.recipients): Exactly one of integrationRef or recipients is required</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>notifyOn</b></td>
        <td>enum</td>
        <td>
          When to notify.<br/>
          <br/>
            <i>Enum</i>: triggeredOnly, triggeredAndResolved<br/>
            <i>Default</i>: triggeredOnly<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupwebhooksindexretriggeringperiod">retriggeringPeriod</a></b></td>
        <td>object</td>
        <td>
          When to re-trigger.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.webhooks[index].integration
<sup><sup>[↩ Parent](#alertspecnotificationgroupwebhooksindex)</sup></sup>



Type and spec of webhook.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupwebhooksindexintegrationintegrationref">integrationRef</a></b></td>
        <td>object</td>
        <td>
          Reference to the webhook.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.backendRef) || has(self.resourceRef): Exactly one of backendRef or resourceRef is required</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>recipients</b></td>
        <td>[]string</td>
        <td>
          Recipients for the notification.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.webhooks[index].integration.integrationRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupwebhooksindexintegration)</sup></sup>



Reference to the webhook.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupwebhooksindexintegrationintegrationrefbackendref">backendRef</a></b></td>
        <td>object</td>
        <td>
          Backend reference for the outbound webhook.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.id) != has(self.name): One of id or name is required</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupwebhooksindexintegrationintegrationrefresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          Resource reference for use with the alert notification.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.webhooks[index].integration.integrationRef.backendRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupwebhooksindexintegrationintegrationref)</sup></sup>



Backend reference for the outbound webhook.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>id</b></td>
        <td>integer</td>
        <td>
          Webhook ID.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the webhook.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.webhooks[index].integration.integrationRef.resourceRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupwebhooksindexintegrationintegrationref)</sup></sup>



Resource reference for use with the alert notification.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.webhooks[index].retriggeringPeriod
<sup><sup>[↩ Parent](#alertspecnotificationgroupwebhooksindex)</sup></sup>



When to re-trigger.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>minutes</b></td>
        <td>integer</td>
        <td>
          Delay between re-triggered alerts.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index]
<sup><sup>[↩ Parent](#alertspec)</sup></sup>



Notification group to use for alert notifications.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexdestinationsindex">destinations</a></b></td>
        <td>[]object</td>
        <td>
          The destinations for notifications (Notification Center feature).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>groupByKeys</b></td>
        <td>[]string</td>
        <td>
          Group notification by these keys.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexrouter">router</a></b></td>
        <td>object</td>
        <td>
          The router for notifications (Notification Center feature).<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexwebhooksindex">webhooks</a></b></td>
        <td>[]object</td>
        <td>
          Webhooks to trigger for notifications.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].destinations[index]
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexdestinationsindexconnector">connector</a></b></td>
        <td>object</td>
        <td>
          Connector is the connector for the destination. Should be one of backendRef or resourceRef.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.backendRef) != has(self.resourceRef): Exactly one of backendRef or resourceRef must be set</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>notifyOn</b></td>
        <td>enum</td>
        <td>
          When to notify.<br/>
          <br/>
            <i>Enum</i>: triggeredOnly, triggeredAndResolved<br/>
            <i>Default</i>: triggeredOnly<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexdestinationsindextriggeredroutingoverrides">triggeredRoutingOverrides</a></b></td>
        <td>object</td>
        <td>
          The routing configuration to override from the connector/preset for triggered notifications.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexdestinationsindexpreset">preset</a></b></td>
        <td>object</td>
        <td>
          Preset is the preset for the destination. Should be one of backendRef or resourceRef.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.backendRef) != has(self.resourceRef): Exactly one of backendRef or resourceRef must be set</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexdestinationsindexresolvedroutingoverrides">resolvedRoutingOverrides</a></b></td>
        <td>object</td>
        <td>
          Optional routing configuration to override from the connector/preset for resolved notifications.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].destinations[index].connector
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexdestinationsindex)</sup></sup>



Connector is the connector for the destination. Should be one of backendRef or resourceRef.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexdestinationsindexconnectorbackendref">backendRef</a></b></td>
        <td>object</td>
        <td>
          BackendRef is a reference to a backend resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexdestinationsindexconnectorresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          ResourceRef is a reference to a Kubernetes resource.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].destinations[index].connector.backendRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexdestinationsindexconnector)</sup></sup>



BackendRef is a reference to a backend resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].destinations[index].connector.resourceRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexdestinationsindexconnector)</sup></sup>



ResourceRef is a reference to a Kubernetes resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].destinations[index].triggeredRoutingOverrides
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexdestinationsindex)</sup></sup>



The routing configuration to override from the connector/preset for triggered notifications.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexdestinationsindextriggeredroutingoverridesconfigoverrides">configOverrides</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].destinations[index].triggeredRoutingOverrides.configOverrides
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexdestinationsindextriggeredroutingoverrides)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>outputSchemaId</b></td>
        <td>string</td>
        <td>
          The ID of the output schema to use for routing notifications<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexdestinationsindextriggeredroutingoverridesconfigoverridesconnectorconfigfieldsindex">connectorConfigFields</a></b></td>
        <td>[]object</td>
        <td>
          Connector configuration fields.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexdestinationsindextriggeredroutingoverridesconfigoverridesmessageconfigfieldsindex">messageConfigFields</a></b></td>
        <td>[]object</td>
        <td>
          Notification message configuration fields.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].destinations[index].triggeredRoutingOverrides.configOverrides.connectorConfigFields[index]
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexdestinationsindextriggeredroutingoverridesconfigoverrides)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldName</b></td>
        <td>string</td>
        <td>
          The name of the configuration field.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>template</b></td>
        <td>string</td>
        <td>
          The template for the configuration field.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].destinations[index].triggeredRoutingOverrides.configOverrides.messageConfigFields[index]
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexdestinationsindextriggeredroutingoverridesconfigoverrides)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldName</b></td>
        <td>string</td>
        <td>
          The name of the configuration field.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>template</b></td>
        <td>string</td>
        <td>
          The template for the configuration field.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].destinations[index].preset
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexdestinationsindex)</sup></sup>



Preset is the preset for the destination. Should be one of backendRef or resourceRef.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexdestinationsindexpresetbackendref">backendRef</a></b></td>
        <td>object</td>
        <td>
          BackendRef is a reference to a backend resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexdestinationsindexpresetresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          ResourceRef is a reference to a Kubernetes resource.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].destinations[index].preset.backendRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexdestinationsindexpreset)</sup></sup>



BackendRef is a reference to a backend resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].destinations[index].preset.resourceRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexdestinationsindexpreset)</sup></sup>



ResourceRef is a reference to a Kubernetes resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].destinations[index].resolvedRoutingOverrides
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexdestinationsindex)</sup></sup>



Optional routing configuration to override from the connector/preset for resolved notifications.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexdestinationsindexresolvedroutingoverridesconfigoverrides">configOverrides</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].destinations[index].resolvedRoutingOverrides.configOverrides
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexdestinationsindexresolvedroutingoverrides)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>outputSchemaId</b></td>
        <td>string</td>
        <td>
          The ID of the output schema to use for routing notifications<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexdestinationsindexresolvedroutingoverridesconfigoverridesconnectorconfigfieldsindex">connectorConfigFields</a></b></td>
        <td>[]object</td>
        <td>
          Connector configuration fields.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexdestinationsindexresolvedroutingoverridesconfigoverridesmessageconfigfieldsindex">messageConfigFields</a></b></td>
        <td>[]object</td>
        <td>
          Notification message configuration fields.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].destinations[index].resolvedRoutingOverrides.configOverrides.connectorConfigFields[index]
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexdestinationsindexresolvedroutingoverridesconfigoverrides)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldName</b></td>
        <td>string</td>
        <td>
          The name of the configuration field.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>template</b></td>
        <td>string</td>
        <td>
          The template for the configuration field.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].destinations[index].resolvedRoutingOverrides.configOverrides.messageConfigFields[index]
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexdestinationsindexresolvedroutingoverridesconfigoverrides)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldName</b></td>
        <td>string</td>
        <td>
          The name of the configuration field.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>template</b></td>
        <td>string</td>
        <td>
          The template for the configuration field.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].router
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindex)</sup></sup>



The router for notifications (Notification Center feature).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>notifyOn</b></td>
        <td>enum</td>
        <td>
          When to notify.<br/>
          <br/>
            <i>Enum</i>: triggeredOnly, triggeredAndResolved<br/>
            <i>Default</i>: triggeredOnly<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].webhooks[index]
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindex)</sup></sup>



Settings for a notification webhook.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexwebhooksindexintegration">integration</a></b></td>
        <td>object</td>
        <td>
          Type and spec of webhook.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.integrationRef) || has(self.recipients): Exactly one of integrationRef or recipients is required</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>notifyOn</b></td>
        <td>enum</td>
        <td>
          When to notify.<br/>
          <br/>
            <i>Enum</i>: triggeredOnly, triggeredAndResolved<br/>
            <i>Default</i>: triggeredOnly<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexwebhooksindexretriggeringperiod">retriggeringPeriod</a></b></td>
        <td>object</td>
        <td>
          When to re-trigger.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].webhooks[index].integration
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexwebhooksindex)</sup></sup>



Type and spec of webhook.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexwebhooksindexintegrationintegrationref">integrationRef</a></b></td>
        <td>object</td>
        <td>
          Reference to the webhook.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.backendRef) || has(self.resourceRef): Exactly one of backendRef or resourceRef is required</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>recipients</b></td>
        <td>[]string</td>
        <td>
          Recipients for the notification.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].webhooks[index].integration.integrationRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexwebhooksindexintegration)</sup></sup>



Reference to the webhook.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexwebhooksindexintegrationintegrationrefbackendref">backendRef</a></b></td>
        <td>object</td>
        <td>
          Backend reference for the outbound webhook.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.id) != has(self.name): One of id or name is required</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexwebhooksindexintegrationintegrationrefresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          Resource reference for use with the alert notification.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].webhooks[index].integration.integrationRef.backendRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexwebhooksindexintegrationintegrationref)</sup></sup>



Backend reference for the outbound webhook.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>id</b></td>
        <td>integer</td>
        <td>
          Webhook ID.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the webhook.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].webhooks[index].integration.integrationRef.resourceRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexwebhooksindexintegrationintegrationref)</sup></sup>



Resource reference for use with the alert notification.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].webhooks[index].retriggeringPeriod
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexwebhooksindex)</sup></sup>



When to re-trigger.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>minutes</b></td>
        <td>integer</td>
        <td>
          Delay between re-triggered alerts.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.schedule
<sup><sup>[↩ Parent](#alertspec)</sup></sup>



Alert activity schedule. Will be activated all the time if not specified.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>timeZone</b></td>
        <td>string</td>
        <td>
          Time zone.<br/>
          <br/>
            <i>Default</i>: UTC+00<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecscheduleactiveon">activeOn</a></b></td>
        <td>object</td>
        <td>
          Schedule to have the alert active.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.schedule.activeOn
<sup><sup>[↩ Parent](#alertspecschedule)</sup></sup>



Schedule to have the alert active.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>dayOfWeek</b></td>
        <td>[]enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: sunday, monday, tuesday, wednesday, thursday, friday, saturday<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>endTime</b></td>
        <td>string</td>
        <td>
          Time of day.<br/>
          <br/>
            <i>Default</i>: 23:59<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>startTime</b></td>
        <td>string</td>
        <td>
          Time of day.<br/>
          <br/>
            <i>Default</i>: 00:00<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.status
<sup><sup>[↩ Parent](#alert)</sup></sup>



AlertStatus defines the observed state of Alert

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.status.conditions[index]
<sup><sup>[↩ Parent](#alertstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

# coralogix.com/v1alpha1

Resource Types:

- [AlertScheduler](#alertscheduler)

- [ApiKey](#apikey)

- [ArchiveLogsTarget](#archivelogstarget)

- [Connector](#connector)

- [CustomRole](#customrole)

- [Dashboard](#dashboard)

- [DashboardsFolder](#dashboardsfolder)

- [Extension](#extension)

- [GlobalRouter](#globalrouter)

- [Group](#group)

- [Integration](#integration)

- [OutboundWebhook](#outboundwebhook)

- [Preset](#preset)

- [RecordingRuleGroupSet](#recordingrulegroupset)

- [RuleGroup](#rulegroup)

- [Scope](#scope)

- [TCOLogsPolicies](#tcologspolicies)

- [TCOTracesPolicies](#tcotracespolicies)

- [ViewFolder](#viewfolder)

- [View](#view)




## AlertScheduler
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






AlertScheduler is the Schema for the alertschedulers API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>AlertScheduler</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#alertschedulerspec">spec</a></b></td>
        <td>object</td>
        <td>
          AlertSchedulerSpec defines the desired state Coralogix AlertScheduler.
It is used to suppress or activate alerts based on a schedule.
See also https://coralogix.com/docs/user-guides/alerting/alert-suppression-rules/

Added in v0.4.0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertschedulerstatus">status</a></b></td>
        <td>object</td>
        <td>
          AlertSchedulerStatus defines the observed state of AlertScheduler.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### AlertScheduler.spec
<sup><sup>[↩ Parent](#alertscheduler)</sup></sup>



AlertSchedulerSpec defines the desired state Coralogix AlertScheduler.
It is used to suppress or activate alerts based on a schedule.
See also https://coralogix.com/docs/user-guides/alerting/alert-suppression-rules/

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertschedulerspecfilter">filter</a></b></td>
        <td>object</td>
        <td>
          Alert Scheduler filter. Exactly one of `metaLabels` or `alerts` can be set.
If none of them set, all alerts will be affected.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.metaLabels) != has(self.alerts): Exactly one of metaLabels or alerts must be set</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Alert Scheduler name.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertschedulerspecschedule">schedule</a></b></td>
        <td>object</td>
        <td>
          Alert Scheduler schedule. Exactly one of `oneTime` or `recurring` must be set.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.oneTime) != has(self.recurring): Exactly one of oneTime or recurring must be set</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Alert Scheduler description.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          Alert Scheduler enabled. If set to `false`, the alert scheduler will be disabled. True by default.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertschedulerspecmetalabelsindex">metaLabels</a></b></td>
        <td>[]object</td>
        <td>
          Alert Scheduler meta labels.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### AlertScheduler.spec.filter
<sup><sup>[↩ Parent](#alertschedulerspec)</sup></sup>



Alert Scheduler filter. Exactly one of `metaLabels` or `alerts` can be set.
If none of them set, all alerts will be affected.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>whatExpression</b></td>
        <td>string</td>
        <td>
          DataPrime query expression - https://coralogix.com/docs/dataprime-query-language.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertschedulerspecfilteralertsindex">alerts</a></b></td>
        <td>[]object</td>
        <td>
          Alert references. Conflicts with `metaLabels`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertschedulerspecfiltermetalabelsindex">metaLabels</a></b></td>
        <td>[]object</td>
        <td>
          Alert Scheduler meta labels. Conflicts with `alerts`.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### AlertScheduler.spec.filter.alerts[index]
<sup><sup>[↩ Parent](#alertschedulerspecfilter)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertschedulerspecfilteralertsindexresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          Alert custom resource name and namespace. If namespace is not set, the AlertScheduler namespace will be used.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### AlertScheduler.spec.filter.alerts[index].resourceRef
<sup><sup>[↩ Parent](#alertschedulerspecfilteralertsindex)</sup></sup>



Alert custom resource name and namespace. If namespace is not set, the AlertScheduler namespace will be used.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource (not id).<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### AlertScheduler.spec.filter.metaLabels[index]
<sup><sup>[↩ Parent](#alertschedulerspecfilter)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### AlertScheduler.spec.schedule
<sup><sup>[↩ Parent](#alertschedulerspec)</sup></sup>



Alert Scheduler schedule. Exactly one of `oneTime` or `recurring` must be set.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>operation</b></td>
        <td>enum</td>
        <td>
          The operation to perform. Can be `mute` or `activate`.<br/>
          <br/>
            <i>Enum</i>: mute, activate<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertschedulerspecscheduleonetime">oneTime</a></b></td>
        <td>object</td>
        <td>
          One-time schedule. Conflicts with `recurring`.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.endTime) != has(self.duration): Exactly one of endTime or duration must be set</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertschedulerspecschedulerecurring">recurring</a></b></td>
        <td>object</td>
        <td>
          Recurring schedule. Conflicts with `oneTime`.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.always) != has(self.dynamic): Exactly one of always or dynamic must be set</li>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### AlertScheduler.spec.schedule.oneTime
<sup><sup>[↩ Parent](#alertschedulerspecschedule)</sup></sup>



One-time schedule. Conflicts with `recurring`.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>startTime</b></td>
        <td>string</td>
        <td>
          The start time of the time frame. In isodate format. For example, `2021-01-01T00:00:00.000`.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>timezone</b></td>
        <td>string</td>
        <td>
          The timezone of the time frame. For example, `UTC-4` or `UTC+10`.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertschedulerspecscheduleonetimeduration">duration</a></b></td>
        <td>object</td>
        <td>
          The duration from the start time to wait before the operation is performed.
Conflicts with `endTime`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>endTime</b></td>
        <td>string</td>
        <td>
          The end time of the time frame. In isodate format. For example, `2021-01-01T00:00:00.000`.
Conflicts with `duration`.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### AlertScheduler.spec.schedule.oneTime.duration
<sup><sup>[↩ Parent](#alertschedulerspecscheduleonetime)</sup></sup>



The duration from the start time to wait before the operation is performed.
Conflicts with `endTime`.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>forOver</b></td>
        <td>integer</td>
        <td>
          The number of time units to wait before the alert is triggered. For example,
if the frequency is set to `hours` and the value is set to `2`, the alert will be triggered after 2 hours.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>frequency</b></td>
        <td>enum</td>
        <td>
          The time unit to wait before the alert is triggered. Can be `minutes`, `hours` or `days`.<br/>
          <br/>
            <i>Enum</i>: minutes, hours, days<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### AlertScheduler.spec.schedule.recurring
<sup><sup>[↩ Parent](#alertschedulerspecschedule)</sup></sup>



Recurring schedule. Conflicts with `oneTime`.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>always</b></td>
        <td>object</td>
        <td>
          Recurring always.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertschedulerspecschedulerecurringdynamic">dynamic</a></b></td>
        <td>object</td>
        <td>
          Dynamic schedule.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### AlertScheduler.spec.schedule.recurring.dynamic
<sup><sup>[↩ Parent](#alertschedulerspecschedulerecurring)</sup></sup>



Dynamic schedule.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertschedulerspecschedulerecurringdynamicfrequency">frequency</a></b></td>
        <td>object</td>
        <td>
          The rule will be activated in a recurring mode (daily, weekly or monthly).<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>repeatEvery</b></td>
        <td>integer</td>
        <td>
          The rule will be activated in a recurring mode according to the interval.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertschedulerspecschedulerecurringdynamictimeframe">timeFrame</a></b></td>
        <td>object</td>
        <td>
          The time frame of the rule.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.endTime) != has(self.duration): Exactly one of endTime or duration must be set</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>terminationDate</b></td>
        <td>string</td>
        <td>
          The termination date of the rule.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### AlertScheduler.spec.schedule.recurring.dynamic.frequency
<sup><sup>[↩ Parent](#alertschedulerspecschedulerecurringdynamic)</sup></sup>



The rule will be activated in a recurring mode (daily, weekly or monthly).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>daily</b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertschedulerspecschedulerecurringdynamicfrequencymonthly">monthly</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertschedulerspecschedulerecurringdynamicfrequencyweekly">weekly</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### AlertScheduler.spec.schedule.recurring.dynamic.frequency.monthly
<sup><sup>[↩ Parent](#alertschedulerspecschedulerecurringdynamicfrequency)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>days</b></td>
        <td>[]integer</td>
        <td>
          The days of the month to activate the rule.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### AlertScheduler.spec.schedule.recurring.dynamic.frequency.weekly
<sup><sup>[↩ Parent](#alertschedulerspecschedulerecurringdynamicfrequency)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>days</b></td>
        <td>[]enum</td>
        <td>
          The days of the week to activate the rule.<br/>
          <br/>
            <i>Enum</i>: Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### AlertScheduler.spec.schedule.recurring.dynamic.timeFrame
<sup><sup>[↩ Parent](#alertschedulerspecschedulerecurringdynamic)</sup></sup>



The time frame of the rule.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>startTime</b></td>
        <td>string</td>
        <td>
          The start time of the time frame. In isodate format. For example, `2021-01-01T00:00:00.000`.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>timezone</b></td>
        <td>string</td>
        <td>
          The timezone of the time frame. For example, `UTC-4` or `UTC+10`.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertschedulerspecschedulerecurringdynamictimeframeduration">duration</a></b></td>
        <td>object</td>
        <td>
          The duration from the start time to wait before the operation is performed.
Conflicts with `endTime`.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>endTime</b></td>
        <td>string</td>
        <td>
          The end time of the time frame. In isodate format. For example, `2021-01-01T00:00:00.000`.
Conflicts with `duration`.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### AlertScheduler.spec.schedule.recurring.dynamic.timeFrame.duration
<sup><sup>[↩ Parent](#alertschedulerspecschedulerecurringdynamictimeframe)</sup></sup>



The duration from the start time to wait before the operation is performed.
Conflicts with `endTime`.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>forOver</b></td>
        <td>integer</td>
        <td>
          The number of time units to wait before the alert is triggered. For example,
if the frequency is set to `hours` and the value is set to `2`, the alert will be triggered after 2 hours.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>frequency</b></td>
        <td>enum</td>
        <td>
          The time unit to wait before the alert is triggered. Can be `minutes`, `hours` or `days`.<br/>
          <br/>
            <i>Enum</i>: minutes, hours, days<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### AlertScheduler.spec.metaLabels[index]
<sup><sup>[↩ Parent](#alertschedulerspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### AlertScheduler.status
<sup><sup>[↩ Parent](#alertscheduler)</sup></sup>



AlertSchedulerStatus defines the observed state of AlertScheduler.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertschedulerstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### AlertScheduler.status.conditions[index]
<sup><sup>[↩ Parent](#alertschedulerstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## ApiKey
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






ApiKey is the Schema for the apikeys API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>ApiKey</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#apikeyspec">spec</a></b></td>
        <td>object</td>
        <td>
          ApiKeySpec defines the desired state of a Coralogix ApiKey.
See also https://coralogix.com/docs/user-guides/account-management/api-keys/api-keys/

Added in v0.4.0<br/>
          <br/>
            <i>Validations</i>:<li>has(self.presets) || has(self.permissions): At least one of presets or permissions must be set</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#apikeystatus">status</a></b></td>
        <td>object</td>
        <td>
          ApiKeyStatus defines the observed state of ApiKey.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ApiKey.spec
<sup><sup>[↩ Parent](#apikey)</sup></sup>



ApiKeySpec defines the desired state of a Coralogix ApiKey.
See also https://coralogix.com/docs/user-guides/account-management/api-keys/api-keys/

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the ApiKey<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#apikeyspecowner">owner</a></b></td>
        <td>object</td>
        <td>
          Owner of the ApiKey.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.userId) != has(self.teamId): Exactly one of userId or teamId must be set</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>active</b></td>
        <td>boolean</td>
        <td>
          Whether the ApiKey Is active.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>permissions</b></td>
        <td>[]string</td>
        <td>
          Permissions of the ApiKey<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>presets</b></td>
        <td>[]string</td>
        <td>
          Permission Presets that the ApiKey uses.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ApiKey.spec.owner
<sup><sup>[↩ Parent](#apikeyspec)</sup></sup>



Owner of the ApiKey.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>teamId</b></td>
        <td>integer</td>
        <td>
          Team that owns the key.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>userId</b></td>
        <td>string</td>
        <td>
          User that owns the key.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ApiKey.status
<sup><sup>[↩ Parent](#apikey)</sup></sup>



ApiKeyStatus defines the observed state of ApiKey.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#apikeystatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ApiKey.status.conditions[index]
<sup><sup>[↩ Parent](#apikeystatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## ArchiveLogsTarget
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>








<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>ArchiveLogsTarget</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#archivelogstargetspec">spec</a></b></td>
        <td>object</td>
        <td>
          Added in version v1.0.0
ArchiveLogsTargetSpec defines the desired state of a Coralogix archive logs target.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.s3Target) != has(self.ibmCosTarget): Exactly one of s3Target or ibmCosTarget must be specified</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#archivelogstargetstatus">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ArchiveLogsTarget.spec
<sup><sup>[↩ Parent](#archivelogstarget)</sup></sup>



Added in version v1.0.0
ArchiveLogsTargetSpec defines the desired state of a Coralogix archive logs target.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#archivelogstargetspecibmcostarget">ibmCosTarget</a></b></td>
        <td>object</td>
        <td>
          The IBM COS target configuration.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#archivelogstargetspecs3target">s3Target</a></b></td>
        <td>object</td>
        <td>
          The S3 target configuration.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ArchiveLogsTarget.spec.ibmCosTarget
<sup><sup>[↩ Parent](#archivelogstargetspec)</sup></sup>



The IBM COS target configuration.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>bucketCrn</b></td>
        <td>string</td>
        <td>
          BucketCrn is the CRN of the IBM COS bucket.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>bucketType</b></td>
        <td>enum</td>
        <td>
          BucketType defines the type of the bucket.<br/>
          <br/>
            <i>Enum</i>: UNSPECIFIED, EXTERNAL, INTERNAL<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>endpoint</b></td>
        <td>string</td>
        <td>
          Endpoint is the endpoint URL for the IBM COS service.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>serviceCrn</b></td>
        <td>string</td>
        <td>
          ServiceCrn is the CRN of the service instance.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ArchiveLogsTarget.spec.s3Target
<sup><sup>[↩ Parent](#archivelogstargetspec)</sup></sup>



The S3 target configuration.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>bucketName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>region</b></td>
        <td>string</td>
        <td>
          The region of the S3 bucket.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ArchiveLogsTarget.status
<sup><sup>[↩ Parent](#archivelogstarget)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#archivelogstargetstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          ID is the identifier of the archive logs target.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ArchiveLogsTarget.status.conditions[index]
<sup><sup>[↩ Parent](#archivelogstargetstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## Connector
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






Connector is the Schema for the connectors API.
NOTE: This CRD exposes a new feature and may have breaking changes in future releases.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Connector</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#connectorspec">spec</a></b></td>
        <td>object</td>
        <td>
          ConnectorSpec defines the desired state of Connector.
See also https://coralogix.com/docs/user-guides/notification-center/introduction/connectors-explained/

Added in v0.4.0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#connectorstatus">status</a></b></td>
        <td>object</td>
        <td>
          ConnectorStatus defines the observed state of Connector.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Connector.spec
<sup><sup>[↩ Parent](#connector)</sup></sup>



ConnectorSpec defines the desired state of Connector.
See also https://coralogix.com/docs/user-guides/notification-center/introduction/connectors-explained/

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#connectorspecconnectorconfig">connectorConfig</a></b></td>
        <td>object</td>
        <td>
          ConnectorConfig is the configuration of the connector.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description is the description of the connector.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is the name of the connector.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>
          Type is the type of the connector. Can be one of slack, genericHttps, or pagerDuty.<br/>
          <br/>
            <i>Enum</i>: slack, genericHttps, pagerDuty<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#connectorspecconfigoverridesindex">configOverrides</a></b></td>
        <td>[]object</td>
        <td>
          ConfigOverrides are the entity type config overrides for the connector.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Connector.spec.connectorConfig
<sup><sup>[↩ Parent](#connectorspec)</sup></sup>



ConnectorConfig is the configuration of the connector.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#connectorspecconnectorconfigfieldsindex">fields</a></b></td>
        <td>[]object</td>
        <td>
          Fields are the fields of the connector config.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Connector.spec.connectorConfig.fields[index]
<sup><sup>[↩ Parent](#connectorspecconnectorconfig)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldName</b></td>
        <td>string</td>
        <td>
          FieldName is the name of the field. e.g. "channel" for slack.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is the value of the field.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Connector.spec.configOverrides[index]
<sup><sup>[↩ Parent](#connectorspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>entityType</b></td>
        <td>enum</td>
        <td>
          EntityType is the entity type for the config override. Should equal "alerts".<br/>
          <br/>
            <i>Enum</i>: alerts<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#connectorspecconfigoverridesindexfieldsindex">fields</a></b></td>
        <td>[]object</td>
        <td>
          Fields are the templated fields for the config override.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Connector.spec.configOverrides[index].fields[index]
<sup><sup>[↩ Parent](#connectorspecconfigoverridesindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldName</b></td>
        <td>string</td>
        <td>
          FieldName is the name of the field. e.g. "channel" for slack.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>template</b></td>
        <td>string</td>
        <td>
          Template is the template for the field.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Connector.status
<sup><sup>[↩ Parent](#connector)</sup></sup>



ConnectorStatus defines the observed state of Connector.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#connectorstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Connector.status.conditions[index]
<sup><sup>[↩ Parent](#connectorstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## CustomRole
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






CustomRole is the Schema for the customroles API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>CustomRole</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#customrolespec">spec</a></b></td>
        <td>object</td>
        <td>
          CustomRoleSpec defines the desired state of a Coralogix Custom Role.
See also https://coralogix.com/docs/user-guides/account-management/user-management/create-roles-and-permissions/

Added in v0.4.0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#customrolestatus">status</a></b></td>
        <td>object</td>
        <td>
          CustomRoleStatus defines the observed state of CustomRole.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### CustomRole.spec
<sup><sup>[↩ Parent](#customrole)</sup></sup>



CustomRoleSpec defines the desired state of a Coralogix Custom Role.
See also https://coralogix.com/docs/user-guides/account-management/user-management/create-roles-and-permissions/

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description of the custom role.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the custom role.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>parentRoleName</b></td>
        <td>string</td>
        <td>
          Parent role name.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>permissions</b></td>
        <td>[]string</td>
        <td>
          Custom role permissions.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### CustomRole.status
<sup><sup>[↩ Parent](#customrole)</sup></sup>



CustomRoleStatus defines the observed state of CustomRole.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#customrolestatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### CustomRole.status.conditions[index]
<sup><sup>[↩ Parent](#customrolestatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## Dashboard
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






Dashboard is the Schema for the dashboards API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Dashboard</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#dashboardspec">spec</a></b></td>
        <td>object</td>
        <td>
          DashboardSpec defines the desired state of Dashboard.
See also https://coralogix.com/docs/user-guides/custom-dashboards/getting-started/

Added in v0.4.0<br/>
          <br/>
            <i>Validations</i>:<li>!(has(self.json) && has(self.configMapRef)): Only one of json or configMapRef can be declared at the same time</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#dashboardstatus">status</a></b></td>
        <td>object</td>
        <td>
          DashboardStatus defines the observed state of Dashboard.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Dashboard.spec
<sup><sup>[↩ Parent](#dashboard)</sup></sup>



DashboardSpec defines the desired state of Dashboard.
See also https://coralogix.com/docs/user-guides/custom-dashboards/getting-started/

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#dashboardspecconfigmapref">configMapRef</a></b></td>
        <td>object</td>
        <td>
          model from configmap<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#dashboardspecfolderref">folderRef</a></b></td>
        <td>object</td>
        <td>
          <br/>
          <br/>
            <i>Validations</i>:<li>has(self.backendRef) || has(self.resourceRef): One of backendRef or resourceRef is required</li><li>!(has(self.backendRef) && has(self.resourceRef)): Only one of backendRef or resourceRef can be declared at the same time</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>gzipJson</b></td>
        <td>string</td>
        <td>
          GzipJson the model's JSON compressed with Gzip. Base64-encoded when in YAML.<br/>
          <br/>
            <i>Format</i>: byte<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>json</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Dashboard.spec.configMapRef
<sup><sup>[↩ Parent](#dashboardspec)</sup></sup>



model from configmap

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>
          The key to select.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the referent.
This field is effectively required, but due to backwards compatibility is
allowed to be empty. Instances of this type with an empty value here are
almost certainly wrong.
More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names<br/>
          <br/>
            <i>Default</i>: <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>
          Specify whether the ConfigMap or its key must be defined<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Dashboard.spec.folderRef
<sup><sup>[↩ Parent](#dashboardspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#dashboardspecfolderrefbackendref">backendRef</a></b></td>
        <td>object</td>
        <td>
          <br/>
          <br/>
            <i>Validations</i>:<li>has(self.id) || has(self.path): One of id or path is required</li><li>!(has(self.id) && has(self.path)): Only one of id or path can be declared at the same time</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#dashboardspecfolderrefresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          Reference to a Coralogix resource within the cluster.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Dashboard.spec.folderRef.backendRef
<sup><sup>[↩ Parent](#dashboardspecfolderref)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          Reference to a folder by its backend's ID.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>
          Reference to a folder by its path (<parent-folder-name-1>/<parent-folder-name-2>/<folder-name>).<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Dashboard.spec.folderRef.resourceRef
<sup><sup>[↩ Parent](#dashboardspecfolderref)</sup></sup>



Reference to a Coralogix resource within the cluster.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource (not id).<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Dashboard.status
<sup><sup>[↩ Parent](#dashboard)</sup></sup>



DashboardStatus defines the observed state of Dashboard.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#dashboardstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Dashboard.status.conditions[index]
<sup><sup>[↩ Parent](#dashboardstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## DashboardsFolder
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






DashboardsFolder is the Schema for the dashboardsfolders API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>DashboardsFolder</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#dashboardsfolderspec">spec</a></b></td>
        <td>object</td>
        <td>
          DashboardsFolderSpec defines the desired state of Dashboard Folder.
See also https://coralogix.com/docs/user-guides/custom-dashboards/getting-started/

Added in v0.4.0<br/>
          <br/>
            <i>Validations</i>:<li>!(has(self.parentFolderId) && has(self.parentFolderRef)): Only one of parentFolderID or parentFolderRef can be declared at the same time</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#dashboardsfolderstatus">status</a></b></td>
        <td>object</td>
        <td>
          DashboardsFolderStatus defines the observed state of DashboardsFolder.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### DashboardsFolder.spec
<sup><sup>[↩ Parent](#dashboardsfolder)</sup></sup>



DashboardsFolderSpec defines the desired state of Dashboard Folder.
See also https://coralogix.com/docs/user-guides/custom-dashboards/getting-started/

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>customId</b></td>
        <td>string</td>
        <td>
          A custom ID for the folder. If not provided, a random UUID will be generated. The custom ID is immutable.<br/>
          <br/>
            <i>Validations</i>:<li>self == oldSelf: spec.customId is immutable</li>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>parentFolderId</b></td>
        <td>string</td>
        <td>
          A reference to an existing folder by its backend's ID.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#dashboardsfolderspecparentfolderref">parentFolderRef</a></b></td>
        <td>object</td>
        <td>
          A reference to an existing DashboardsFolder CR.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### DashboardsFolder.spec.parentFolderRef
<sup><sup>[↩ Parent](#dashboardsfolderspec)</sup></sup>



A reference to an existing DashboardsFolder CR.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource (not id).<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### DashboardsFolder.status
<sup><sup>[↩ Parent](#dashboardsfolder)</sup></sup>



DashboardsFolderStatus defines the observed state of DashboardsFolder.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#dashboardsfolderstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### DashboardsFolder.status.conditions[index]
<sup><sup>[↩ Parent](#dashboardsfolderstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## Extension
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






Extension is the Schema for the extensions API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Extension</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#extensionspec">spec</a></b></td>
        <td>object</td>
        <td>
          ExtensionSpec defines the desired state of a Coralogix extension.
See also https://coralogix.com/docs/user-guides/getting-started/packages-and-extensions/integration-packages/<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#extensionstatus">status</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Extension.spec
<sup><sup>[↩ Parent](#extension)</sup></sup>



ExtensionSpec defines the desired state of a Coralogix extension.
See also https://coralogix.com/docs/user-guides/getting-started/packages-and-extensions/integration-packages/

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          Id of the extension to deploy.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>version</b></td>
        <td>string</td>
        <td>
          Desired version of the extension.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>itemIds</b></td>
        <td>[]string</td>
        <td>
          Item IDs to be used by the extension.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Extension.status
<sup><sup>[↩ Parent](#extension)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#extensionstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Extension.status.conditions[index]
<sup><sup>[↩ Parent](#extensionstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## GlobalRouter
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






GlobalRouter is the Schema for the globalrouters API.
NOTE: This CRD exposes a new feature and may have breaking changes in future releases.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>GlobalRouter</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#globalrouterspec">spec</a></b></td>
        <td>object</td>
        <td>
          GlobalRouterSpec defines the desired state of the Global Router.
See also https://coralogix.com/docs/user-guides/notification-center/routing/

Added in v0.4.0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#globalrouterstatus">status</a></b></td>
        <td>object</td>
        <td>
          GlobalRouterStatus defines the observed state of GlobalRouter.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GlobalRouter.spec
<sup><sup>[↩ Parent](#globalrouter)</sup></sup>



GlobalRouterSpec defines the desired state of the Global Router.
See also https://coralogix.com/docs/user-guides/notification-center/routing/

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description is the description of the global router.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>entityType</b></td>
        <td>enum</td>
        <td>
          EntityType is the entity type for the global router. Should equal "alerts".<br/>
          <br/>
            <i>Enum</i>: alerts<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is the name of the global router.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>entityLabels</b></td>
        <td>map[string]string</td>
        <td>
          EntityLabels are optional labels to attach to the global router.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#globalrouterspecfallbackindex">fallback</a></b></td>
        <td>[]object</td>
        <td>
          Fallback is the fallback routing target for the global router.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#globalrouterspecrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          Rules are the routing rules for the global router.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GlobalRouter.spec.fallback[index]
<sup><sup>[↩ Parent](#globalrouterspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#globalrouterspecfallbackindexconnector">connector</a></b></td>
        <td>object</td>
        <td>
          Connector is the connector for the routing target. Should be one of backendRef or resourceRef.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.backendRef) != has(self.resourceRef): Exactly one of backendRef or resourceRef must be set</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>customDetails</b></td>
        <td>map[string]string</td>
        <td>
          CustomDetails are optional custom details to attach to the routing target.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#globalrouterspecfallbackindexpreset">preset</a></b></td>
        <td>object</td>
        <td>
          Preset is the preset for the routing target. Should be one of backendRef or resourceRef.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.backendRef) != has(self.resourceRef): Exactly one of backendRef or resourceRef must be set</li>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GlobalRouter.spec.fallback[index].connector
<sup><sup>[↩ Parent](#globalrouterspecfallbackindex)</sup></sup>



Connector is the connector for the routing target. Should be one of backendRef or resourceRef.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#globalrouterspecfallbackindexconnectorbackendref">backendRef</a></b></td>
        <td>object</td>
        <td>
          BackendRef is a reference to a backend resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#globalrouterspecfallbackindexconnectorresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          ResourceRef is a reference to a Kubernetes resource.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GlobalRouter.spec.fallback[index].connector.backendRef
<sup><sup>[↩ Parent](#globalrouterspecfallbackindexconnector)</sup></sup>



BackendRef is a reference to a backend resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### GlobalRouter.spec.fallback[index].connector.resourceRef
<sup><sup>[↩ Parent](#globalrouterspecfallbackindexconnector)</sup></sup>



ResourceRef is a reference to a Kubernetes resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource (not id).<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GlobalRouter.spec.fallback[index].preset
<sup><sup>[↩ Parent](#globalrouterspecfallbackindex)</sup></sup>



Preset is the preset for the routing target. Should be one of backendRef or resourceRef.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#globalrouterspecfallbackindexpresetbackendref">backendRef</a></b></td>
        <td>object</td>
        <td>
          BackendRef is a reference to a backend resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#globalrouterspecfallbackindexpresetresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          ResourceRef is a reference to a Kubernetes resource.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GlobalRouter.spec.fallback[index].preset.backendRef
<sup><sup>[↩ Parent](#globalrouterspecfallbackindexpreset)</sup></sup>



BackendRef is a reference to a backend resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### GlobalRouter.spec.fallback[index].preset.resourceRef
<sup><sup>[↩ Parent](#globalrouterspecfallbackindexpreset)</sup></sup>



ResourceRef is a reference to a Kubernetes resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource (not id).<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GlobalRouter.spec.rules[index]
<sup><sup>[↩ Parent](#globalrouterspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>condition</b></td>
        <td>string</td>
        <td>
          Condition is the condition for the routing rule.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is the name of the routing rule.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#globalrouterspecrulesindextargetsindex">targets</a></b></td>
        <td>[]object</td>
        <td>
          Targets are the routing targets for the routing rule.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>customDetails</b></td>
        <td>map[string]string</td>
        <td>
          CustomDetails are optional custom details to attach to the routing rule.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GlobalRouter.spec.rules[index].targets[index]
<sup><sup>[↩ Parent](#globalrouterspecrulesindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#globalrouterspecrulesindextargetsindexconnector">connector</a></b></td>
        <td>object</td>
        <td>
          Connector is the connector for the routing target. Should be one of backendRef or resourceRef.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.backendRef) != has(self.resourceRef): Exactly one of backendRef or resourceRef must be set</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>customDetails</b></td>
        <td>map[string]string</td>
        <td>
          CustomDetails are optional custom details to attach to the routing target.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#globalrouterspecrulesindextargetsindexpreset">preset</a></b></td>
        <td>object</td>
        <td>
          Preset is the preset for the routing target. Should be one of backendRef or resourceRef.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.backendRef) != has(self.resourceRef): Exactly one of backendRef or resourceRef must be set</li>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GlobalRouter.spec.rules[index].targets[index].connector
<sup><sup>[↩ Parent](#globalrouterspecrulesindextargetsindex)</sup></sup>



Connector is the connector for the routing target. Should be one of backendRef or resourceRef.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#globalrouterspecrulesindextargetsindexconnectorbackendref">backendRef</a></b></td>
        <td>object</td>
        <td>
          BackendRef is a reference to a backend resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#globalrouterspecrulesindextargetsindexconnectorresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          ResourceRef is a reference to a Kubernetes resource.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GlobalRouter.spec.rules[index].targets[index].connector.backendRef
<sup><sup>[↩ Parent](#globalrouterspecrulesindextargetsindexconnector)</sup></sup>



BackendRef is a reference to a backend resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### GlobalRouter.spec.rules[index].targets[index].connector.resourceRef
<sup><sup>[↩ Parent](#globalrouterspecrulesindextargetsindexconnector)</sup></sup>



ResourceRef is a reference to a Kubernetes resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource (not id).<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GlobalRouter.spec.rules[index].targets[index].preset
<sup><sup>[↩ Parent](#globalrouterspecrulesindextargetsindex)</sup></sup>



Preset is the preset for the routing target. Should be one of backendRef or resourceRef.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#globalrouterspecrulesindextargetsindexpresetbackendref">backendRef</a></b></td>
        <td>object</td>
        <td>
          BackendRef is a reference to a backend resource.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#globalrouterspecrulesindextargetsindexpresetresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          ResourceRef is a reference to a Kubernetes resource.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GlobalRouter.spec.rules[index].targets[index].preset.backendRef
<sup><sup>[↩ Parent](#globalrouterspecrulesindextargetsindexpreset)</sup></sup>



BackendRef is a reference to a backend resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### GlobalRouter.spec.rules[index].targets[index].preset.resourceRef
<sup><sup>[↩ Parent](#globalrouterspecrulesindextargetsindexpreset)</sup></sup>



ResourceRef is a reference to a Kubernetes resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource (not id).<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GlobalRouter.status
<sup><sup>[↩ Parent](#globalrouter)</sup></sup>



GlobalRouterStatus defines the observed state of GlobalRouter.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#globalrouterstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### GlobalRouter.status.conditions[index]
<sup><sup>[↩ Parent](#globalrouterstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## Group
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






Group is the Schema for the groups API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Group</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#groupspec">spec</a></b></td>
        <td>object</td>
        <td>
          GroupSpec defines the desired state of Coralogix Group.
See also https://coralogix.com/docs/user-guides/account-management/user-management/assign-user-roles-and-scopes-via-groups/

Added in v0.4.0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#groupstatus">status</a></b></td>
        <td>object</td>
        <td>
          GroupStatus defines the observed state of Group.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Group.spec
<sup><sup>[↩ Parent](#group)</sup></sup>



GroupSpec defines the desired state of Coralogix Group.
See also https://coralogix.com/docs/user-guides/account-management/user-management/assign-user-roles-and-scopes-via-groups/

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#groupspeccustomrolesindex">customRoles</a></b></td>
        <td>[]object</td>
        <td>
          Custom roles applied to the group.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the group.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description of the group.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#groupspecmembersindex">members</a></b></td>
        <td>[]object</td>
        <td>
          Members of the group.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#groupspecscope">scope</a></b></td>
        <td>object</td>
        <td>
          Scope attached to the group.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Group.spec.customRoles[index]
<sup><sup>[↩ Parent](#groupspec)</sup></sup>



Custom role reference.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#groupspeccustomrolesindexresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          Reference to the custom role within the cluster.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Group.spec.customRoles[index].resourceRef
<sup><sup>[↩ Parent](#groupspeccustomrolesindex)</sup></sup>



Reference to the custom role within the cluster.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource (not id).<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Group.spec.members[index]
<sup><sup>[↩ Parent](#groupspec)</sup></sup>



User on Coralogix.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>userName</b></td>
        <td>string</td>
        <td>
          User's name.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Group.spec.scope
<sup><sup>[↩ Parent](#groupspec)</sup></sup>



Scope attached to the group.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#groupspecscoperesourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          Scope reference.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Group.spec.scope.resourceRef
<sup><sup>[↩ Parent](#groupspecscope)</sup></sup>



Scope reference.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource (not id).<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Group.status
<sup><sup>[↩ Parent](#group)</sup></sup>



GroupStatus defines the observed state of Group.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#groupstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Group.status.conditions[index]
<sup><sup>[↩ Parent](#groupstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## Integration
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






Integration is the Schema for the integrations API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Integration</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#integrationspec">spec</a></b></td>
        <td>object</td>
        <td>
          IntegrationSpec defines the desired state of a Coralogix (managed) integration.
See also https://coralogix.com/docs/user-guides/getting-started/packages-and-extensions/integration-packages/

For available integrations see https://coralogix.com/docs/developer-portal/infrastructure-as-code/terraform-provider/integrations/aws-metrics-collector/ or at https://github.com/coralogix/coralogix-operator/tree/main/config/samples/v1alpha1/integrations.

Added in v0.4.0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#integrationstatus">status</a></b></td>
        <td>object</td>
        <td>
          IntegrationStatus defines the observed state of Integration.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Integration.spec
<sup><sup>[↩ Parent](#integration)</sup></sup>



IntegrationSpec defines the desired state of a Coralogix (managed) integration.
See also https://coralogix.com/docs/user-guides/getting-started/packages-and-extensions/integration-packages/

For available integrations see https://coralogix.com/docs/developer-portal/infrastructure-as-code/terraform-provider/integrations/aws-metrics-collector/ or at https://github.com/coralogix/coralogix-operator/tree/main/config/samples/v1alpha1/integrations.

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>integrationKey</b></td>
        <td>string</td>
        <td>
          Unique name of the integration.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>parameters</b></td>
        <td>object</td>
        <td>
          Parameters required by the integration.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>version</b></td>
        <td>string</td>
        <td>
          Desired version of the integration<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Integration.status
<sup><sup>[↩ Parent](#integration)</sup></sup>



IntegrationStatus defines the observed state of Integration.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#integrationstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Integration.status.conditions[index]
<sup><sup>[↩ Parent](#integrationstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## OutboundWebhook
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






OutboundWebhook is the Schema for the API

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>OutboundWebhook</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspec">spec</a></b></td>
        <td>object</td>
        <td>
          OutboundWebhookSpec defines the desired state of an outbound webhook.
See also https://coralogix.com/docs/user-guides/alerting/outbound-webhooks/aws-eventbridge-outbound-webhook/

Added in v0.4.0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookstatus">status</a></b></td>
        <td>object</td>
        <td>
          OutboundWebhookStatus defines the observed state of OutboundWebhook<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec
<sup><sup>[↩ Parent](#outboundwebhook)</sup></sup>



OutboundWebhookSpec defines the desired state of an outbound webhook.
See also https://coralogix.com/docs/user-guides/alerting/outbound-webhooks/aws-eventbridge-outbound-webhook/

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the webhook.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktype">outboundWebhookType</a></b></td>
        <td>object</td>
        <td>
          Type of webhook.<br/>
          <br/>
            <i>Validations</i>:<li>(has(self.genericWebhook) ? 1 : 0) + (has(self.slack) ? 1 : 0) + (has(self.pagerDuty) ? 1 : 0) + (has(self.sendLog) ? 1 : 0) + (has(self.emailGroup) ? 1 : 0) + (has(self.microsoftTeams) ? 1 : 0) + (has(self.jira) ? 1 : 0) + (has(self.opsgenie) ? 1 : 0) + (has(self.demisto) ? 1 : 0) + (has(self.awsEventBridge) ? 1 : 0) == 1: Exactly one of the following fields must be set: genericWebhook, slack, pagerDuty, sendLog, emailGroup, microsoftTeams, jira, opsgenie, demisto, awsEventBridge</li>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType
<sup><sup>[↩ Parent](#outboundwebhookspec)</sup></sup>



Type of webhook.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypeawseventbridge">awsEventBridge</a></b></td>
        <td>object</td>
        <td>
          AWS eventbridge message.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypedemisto">demisto</a></b></td>
        <td>object</td>
        <td>
          Demisto notification.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypeemailgroup">emailGroup</a></b></td>
        <td>object</td>
        <td>
          Email notification.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypegenericwebhook">genericWebhook</a></b></td>
        <td>object</td>
        <td>
          Generic HTTP(s) webhook.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypejira">jira</a></b></td>
        <td>object</td>
        <td>
          Jira issue.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypemicrosoftteams">microsoftTeams</a></b></td>
        <td>object</td>
        <td>
          Teams message.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypeopsgenie">opsgenie</a></b></td>
        <td>object</td>
        <td>
          Opsgenie notification.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypepagerduty">pagerDuty</a></b></td>
        <td>object</td>
        <td>
          PagerDuty notification.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypesendlog">sendLog</a></b></td>
        <td>object</td>
        <td>
          SendLog notification.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypeslack">slack</a></b></td>
        <td>object</td>
        <td>
          Slack message.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.awsEventBridge
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>



AWS eventbridge message.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>detail</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>detailType</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>eventBusArn</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>roleName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>source</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.demisto
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>



Demisto notification.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>payload</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>url</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>uuid</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.emailGroup
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>



Email notification.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>emailAddresses</b></td>
        <td>[]string</td>
        <td>
          Recipients<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.genericWebhook
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>



Generic HTTP(s) webhook.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>method</b></td>
        <td>enum</td>
        <td>
          HTTP Method to use.<br/>
          <br/>
            <i>Enum</i>: Unkown, Get, Post, Put<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>url</b></td>
        <td>string</td>
        <td>
          URL to call<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>headers</b></td>
        <td>map[string]string</td>
        <td>
          Attached HTTP headers.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>payload</b></td>
        <td>string</td>
        <td>
          Payload of the webhook call.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.jira
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>



Jira issue.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>apiToken</b></td>
        <td>string</td>
        <td>
          API token<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>email</b></td>
        <td>string</td>
        <td>
          Email address associated with the token<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>projectKey</b></td>
        <td>string</td>
        <td>
          Project to add it to.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>url</b></td>
        <td>string</td>
        <td>
          Jira URL<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.microsoftTeams
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>



Teams message.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>url</b></td>
        <td>string</td>
        <td>
          Teams URL<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.opsgenie
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>



Opsgenie notification.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>url</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.pagerDuty
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>



PagerDuty notification.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>serviceKey</b></td>
        <td>string</td>
        <td>
          PagerDuty service key.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.sendLog
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>



SendLog notification.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>payload</b></td>
        <td>string</td>
        <td>
          Payload of the notification<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>url</b></td>
        <td>string</td>
        <td>
          Sendlog URL.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.slack
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>



Slack message.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>url</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypeslackattachmentsindex">attachments</a></b></td>
        <td>[]object</td>
        <td>
          Attachments of the message.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypeslackdigestsindex">digests</a></b></td>
        <td>[]object</td>
        <td>
          Digest configuration.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.slack.attachments[index]
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktypeslack)</sup></sup>



Slack attachment

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>isActive</b></td>
        <td>boolean</td>
        <td>
          Active status.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Attachment to the message.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.slack.digests[index]
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktypeslack)</sup></sup>



Digest config.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>isActive</b></td>
        <td>boolean</td>
        <td>
          Active status.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          Type of digest to send<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OutboundWebhook.status
<sup><sup>[↩ Parent](#outboundwebhook)</sup></sup>



OutboundWebhookStatus defines the observed state of OutboundWebhook

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#outboundwebhookstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>externalId</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OutboundWebhook.status.conditions[index]
<sup><sup>[↩ Parent](#outboundwebhookstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## Preset
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






Preset is the Schema for the presets API.
NOTE: This CRD exposes a new feature and may have breaking changes in future releases.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Preset</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#presetspec">spec</a></b></td>
        <td>object</td>
        <td>
          PresetSpec defines the desired state of Preset.
See also https://coralogix.com/docs/user-guides/notification-center/presets/introduction/

Added in v0.4.0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#presetstatus">status</a></b></td>
        <td>object</td>
        <td>
          PresetStatus defines the observed state of Preset.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Preset.spec
<sup><sup>[↩ Parent](#preset)</sup></sup>



PresetSpec defines the desired state of Preset.
See also https://coralogix.com/docs/user-guides/notification-center/presets/introduction/

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>connectorType</b></td>
        <td>enum</td>
        <td>
          ConnectorType is the type of the connector. Can be one of slack, genericHttps, or pagerDuty.<br/>
          <br/>
            <i>Enum</i>: slack, genericHttps, pagerDuty<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description is the description of the preset.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>entityType</b></td>
        <td>enum</td>
        <td>
          EntityType is the entity type for the preset. Should equal "alerts".<br/>
          <br/>
            <i>Enum</i>: alerts<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is the name of the preset.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#presetspecconfigoverridesindex">configOverrides</a></b></td>
        <td>[]object</td>
        <td>
          ConfigOverrides are the entity type configs, allowing entity type templating.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>parentId</b></td>
        <td>string</td>
        <td>
          ParentId is the ID of the parent preset. For example, "preset_system_slack_alerts_basic".<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Preset.spec.configOverrides[index]
<sup><sup>[↩ Parent](#presetspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#presetspecconfigoverridesindexconditiontype">conditionType</a></b></td>
        <td>object</td>
        <td>
          ConditionType is the condition type for the config override.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.matchEntityType) != has(self.matchEntityTypeAndSubType): exactly one of matchEntityType or matchEntityTypeAndSubType must be set</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#presetspecconfigoverridesindexmessageconfig">messageConfig</a></b></td>
        <td>object</td>
        <td>
          MessageConfig is the message config for the config override.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>payloadType</b></td>
        <td>string</td>
        <td>
          PayloadType is the payload type for the config override.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Preset.spec.configOverrides[index].conditionType
<sup><sup>[↩ Parent](#presetspecconfigoverridesindex)</sup></sup>



ConditionType is the condition type for the config override.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>matchEntityType</b></td>
        <td>object</td>
        <td>
          MatchEntityType is used for matching entity types.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#presetspecconfigoverridesindexconditiontypematchentitytypeandsubtype">matchEntityTypeAndSubType</a></b></td>
        <td>object</td>
        <td>
          MatchEntityTypeAndSubType is used for matching entity subtypes.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Preset.spec.configOverrides[index].conditionType.matchEntityTypeAndSubType
<sup><sup>[↩ Parent](#presetspecconfigoverridesindexconditiontype)</sup></sup>



MatchEntityTypeAndSubType is used for matching entity subtypes.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>entitySubType</b></td>
        <td>string</td>
        <td>
          EntitySubType is the entity subtype for the config override. For example, "logsImmediateTriggered".<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Preset.spec.configOverrides[index].messageConfig
<sup><sup>[↩ Parent](#presetspecconfigoverridesindex)</sup></sup>



MessageConfig is the message config for the config override.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#presetspecconfigoverridesindexmessageconfigfieldsindex">fields</a></b></td>
        <td>[]object</td>
        <td>
          Fields are the fields of the message config.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Preset.spec.configOverrides[index].messageConfig.fields[index]
<sup><sup>[↩ Parent](#presetspecconfigoverridesindexmessageconfig)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldName</b></td>
        <td>string</td>
        <td>
          FieldName is the name of the field. e.g. "title" for slack.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>template</b></td>
        <td>string</td>
        <td>
          Template is the template for the field.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Preset.status
<sup><sup>[↩ Parent](#preset)</sup></sup>



PresetStatus defines the observed state of Preset.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#presetstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Preset.status.conditions[index]
<sup><sup>[↩ Parent](#presetstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## RecordingRuleGroupSet
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






RecordingRuleGroupSet is the Schema for the RecordingRuleGroupSets API

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>RecordingRuleGroupSet</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#recordingrulegroupsetspec">spec</a></b></td>
        <td>object</td>
        <td>
          RecordingRuleGroupSetSpec defines the desired state of a set of Coralogix recording rule groups.
See also https://coralogix.com/docs/user-guides/data-transformation/metric-rules/recording-rules/

Added in v0.4.0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#recordingrulegroupsetstatus">status</a></b></td>
        <td>object</td>
        <td>
          RecordingRuleGroupSetStatus defines the observed state of RecordingRuleGroupSet<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RecordingRuleGroupSet.spec
<sup><sup>[↩ Parent](#recordingrulegroupset)</sup></sup>



RecordingRuleGroupSetSpec defines the desired state of a set of Coralogix recording rule groups.
See also https://coralogix.com/docs/user-guides/data-transformation/metric-rules/recording-rules/

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#recordingrulegroupsetspecgroupsindex">groups</a></b></td>
        <td>[]object</td>
        <td>
          Recording rule groups.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RecordingRuleGroupSet.spec.groups[index]
<sup><sup>[↩ Parent](#recordingrulegroupsetspec)</sup></sup>



A Coralogix recording rule group.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>intervalSeconds</b></td>
        <td>integer</td>
        <td>
          How often rules in the group are evaluated (in seconds).<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 60<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limit</b></td>
        <td>integer</td>
        <td>
          Limits the number of alerts an alerting rule and series a recording-rule can produce. 0 is no limit.<br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          The (unique) rule group name.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#recordingrulegroupsetspecgroupsindexrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          Rules of this group.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RecordingRuleGroupSet.spec.groups[index].rules[index]
<sup><sup>[↩ Parent](#recordingrulegroupsetspecgroupsindex)</sup></sup>



A recording rule.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>expr</b></td>
        <td>string</td>
        <td>
          The PromQL expression to evaluate.
Every evaluation cycle this is evaluated at the current time, and the result recorded as a new set of time series with the metric name as given by 'record'.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>labels</b></td>
        <td>map[string]string</td>
        <td>
          Labels to add or overwrite before storing the result.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>record</b></td>
        <td>string</td>
        <td>
          The name of the time series to output to. Must be a valid metric name.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RecordingRuleGroupSet.status
<sup><sup>[↩ Parent](#recordingrulegroupset)</sup></sup>



RecordingRuleGroupSetStatus defines the observed state of RecordingRuleGroupSet

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#recordingrulegroupsetstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RecordingRuleGroupSet.status.conditions[index]
<sup><sup>[↩ Parent](#recordingrulegroupsetstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## RuleGroup
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






RuleGroup is the Schema for the rulegroups API

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>RuleGroup</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#rulegroupspec">spec</a></b></td>
        <td>object</td>
        <td>
          RuleGroupSpec defines the Desired state of RuleGroup<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupstatus">status</a></b></td>
        <td>object</td>
        <td>
          RuleGroupStatus defines the observed state of RuleGroup<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RuleGroup.spec
<sup><sup>[↩ Parent](#rulegroup)</sup></sup>



RuleGroupSpec defines the Desired state of RuleGroup

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the rule-group.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>active</b></td>
        <td>boolean</td>
        <td>
          Whether the rule-group is active.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>applications</b></td>
        <td>[]string</td>
        <td>
          Rules will execute on logs that match the these applications.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>creator</b></td>
        <td>string</td>
        <td>
          Rule-group creator<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description of the rule-group.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hidden</b></td>
        <td>boolean</td>
        <td>
          Hides the rule-group.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>order</b></td>
        <td>integer</td>
        <td>
          The index of the rule-group between the other rule-groups.<br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severities</b></td>
        <td>[]enum</td>
        <td>
          Rules will execute on logs that match the these severities.<br/>
          <br/>
            <i>Enum</i>: Debug, Verbose, Info, Warning, Error, Critical<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindex">subgroups</a></b></td>
        <td>[]object</td>
        <td>
          Rules within the same subgroup have an OR relationship,
while rules in different subgroups have an AND relationship.
Refer to https://github.com/coralogix/coralogix-operator/blob/main/config/samples/v1alpha1/rulegroups/mixed_rulegroup.yaml
for an example.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subsystems</b></td>
        <td>[]string</td>
        <td>
          Rules will execute on logs that match the these subsystems.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index]
<sup><sup>[↩ Parent](#rulegroupspec)</sup></sup>



Sub group of rules.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>active</b></td>
        <td>boolean</td>
        <td>
          Determines whether to rule will be active or not.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          The rule id.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>order</b></td>
        <td>integer</td>
        <td>
          Determines the index of the rule inside the rule-subgroup.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          List of rules associated with the sub group.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index]
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindex)</sup></sup>



A rule to change data extraction.
See also https://coralogix.com/docs/user-guides/data-transformation/metric-rules/recording-rules/

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the rule.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>active</b></td>
        <td>boolean</td>
        <td>
          Whether the rule will be activated.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexblock">block</a></b></td>
        <td>object</td>
        <td>
          Block rules allow for refined filtering of incoming logs with a Regular Expression.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description of the rule.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexextract">extract</a></b></td>
        <td>object</td>
        <td>
          Use a named Regular Expression group to extract specific values you need as JSON getKeysStrings without having to parse the entire log.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexextracttimestamp">extractTimestamp</a></b></td>
        <td>object</td>
        <td>
          Replace rules are used to replace logs timestamp with JSON field.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexjsonextract">jsonExtract</a></b></td>
        <td>object</td>
        <td>
          Name a JSON field to extract its value directly into a Coralogix metadata field<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexjsonstringify">jsonStringify</a></b></td>
        <td>object</td>
        <td>
          Convert JSON object to JSON string.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexparse">parse</a></b></td>
        <td>object</td>
        <td>
          Parse unstructured logs into JSON format using named Regular Expression groups.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexparsejsonfield">parseJsonField</a></b></td>
        <td>object</td>
        <td>
          Convert JSON string to JSON object.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexremovefields">removeFields</a></b></td>
        <td>object</td>
        <td>
          Remove Fields allows to select fields that will not be indexed.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexreplace">replace</a></b></td>
        <td>object</td>
        <td>
          Replace rules are used to strings in order to fix log structure, change log severity, or obscure information.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].block
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>



Block rules allow for refined filtering of incoming logs with a Regular Expression.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>regex</b></td>
        <td>string</td>
        <td>
          Regular Expression. More info: https://coralogix.com/blog/regex-101/<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>sourceField</b></td>
        <td>string</td>
        <td>
          The field on which the Regular Expression will operate on.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>blockingAllMatchingBlocks</b></td>
        <td>boolean</td>
        <td>
          Block Logic. If true or nor set - blocking all matching blocks, if false - blocking all non-matching blocks.<br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>keepBlockedLogs</b></td>
        <td>boolean</td>
        <td>
          Determines if to view blocked logs in LiveTail and archive to S3.<br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].extract
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>



Use a named Regular Expression group to extract specific values you need as JSON getKeysStrings without having to parse the entire log.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>regex</b></td>
        <td>string</td>
        <td>
          Regular Expression. More info: https://coralogix.com/blog/regex-101/<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>sourceField</b></td>
        <td>string</td>
        <td>
          The field on which the Regular Expression will operate on.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].extractTimestamp
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>



Replace rules are used to replace logs timestamp with JSON field.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldFormatStandard</b></td>
        <td>enum</td>
        <td>
          The format standard to parse the timestamp.<br/>
          <br/>
            <i>Enum</i>: Strftime, JavaSDF, Golang, SecondTS, MilliTS, MicroTS, NanoTS<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>sourceField</b></td>
        <td>string</td>
        <td>
          The field on which the Regular Expression will operate on.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>timeFormat</b></td>
        <td>string</td>
        <td>
          A time formatting string that matches the field format standard.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].jsonExtract
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>



Name a JSON field to extract its value directly into a Coralogix metadata field

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>destinationField</b></td>
        <td>enum</td>
        <td>
          The field that will be populated by the results of the Regular Expression operation.<br/>
          <br/>
            <i>Enum</i>: Category, CLASSNAME, METHODNAME, THREADID, SEVERITY<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>jsonKey</b></td>
        <td>string</td>
        <td>
          JSON key to extract its value directly into a Coralogix metadata field.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].jsonStringify
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>



Convert JSON object to JSON string.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>destinationField</b></td>
        <td>string</td>
        <td>
          The field that will be populated by the results of the Regular Expression<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>sourceField</b></td>
        <td>string</td>
        <td>
          The field on which the Regular Expression will operate on.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>keepSourceField</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].parse
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>



Parse unstructured logs into JSON format using named Regular Expression groups.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>destinationField</b></td>
        <td>string</td>
        <td>
          The field that will be populated by the results of the Regular Expression operation.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>regex</b></td>
        <td>string</td>
        <td>
          Regular Expression. More info: https://coralogix.com/blog/regex-101/<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>sourceField</b></td>
        <td>string</td>
        <td>
          The field on which the Regular Expression will operate on.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].parseJsonField
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>



Convert JSON string to JSON object.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>destinationField</b></td>
        <td>string</td>
        <td>
          The field that will be populated by the results of the Regular Expression<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>keepDestinationField</b></td>
        <td>boolean</td>
        <td>
          Determines whether to keep or to delete the destination field.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>keepSourceField</b></td>
        <td>boolean</td>
        <td>
          Determines whether to keep or to delete the source field.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>sourceField</b></td>
        <td>string</td>
        <td>
          The field on which the Regular Expression will operate on.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].removeFields
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>



Remove Fields allows to select fields that will not be indexed.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>excludedFields</b></td>
        <td>[]string</td>
        <td>
          Excluded fields won't be indexed.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].replace
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>



Replace rules are used to strings in order to fix log structure, change log severity, or obscure information.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>destinationField</b></td>
        <td>string</td>
        <td>
          The field that will be populated by the results of the Regular Expression operation.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>regex</b></td>
        <td>string</td>
        <td>
          Regular Expression. More info: https://coralogix.com/blog/regex-101/<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>replacementString</b></td>
        <td>string</td>
        <td>
          The string that will replace the matched Regular Expression<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>sourceField</b></td>
        <td>string</td>
        <td>
          The field on which the Regular Expression will operate on.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RuleGroup.status
<sup><sup>[↩ Parent](#rulegroup)</sup></sup>



RuleGroupStatus defines the observed state of RuleGroup

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#rulegroupstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RuleGroup.status.conditions[index]
<sup><sup>[↩ Parent](#rulegroupstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## Scope
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






Scope is the Schema for the scopes API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Scope</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#scopespec">spec</a></b></td>
        <td>object</td>
        <td>
          ScopeSpec defines the desired state of a Coralogix Scope.
See also https://coralogix.com/docs/user-guides/account-management/user-management/scopes/

Added in v0.4.0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#scopestatus">status</a></b></td>
        <td>object</td>
        <td>
          ScopeStatus defines the observed state of Coralogix Scope.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Scope.spec
<sup><sup>[↩ Parent](#scope)</sup></sup>



ScopeSpec defines the desired state of a Coralogix Scope.
See also https://coralogix.com/docs/user-guides/account-management/user-management/scopes/

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>defaultExpression</b></td>
        <td>enum</td>
        <td>
          Default expression to use when no filter matches the query. Until further notice, this is limited to `true` (everything is included) or `false` (nothing is included). Use a version tag (e.g `<v1>true` or `<v1>false`)<br/>
          <br/>
            <i>Enum</i>: <v1>true, <v1>false<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#scopespecfiltersindex">filters</a></b></td>
        <td>[]object</td>
        <td>
          Filters applied to include data in the scope.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Scope display name.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description of the scope. Optional.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Scope.spec.filters[index]
<sup><sup>[↩ Parent](#scopespec)</sup></sup>



ScopeFilter defines a filter to include data in a scope.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>entityType</b></td>
        <td>enum</td>
        <td>
          Entity type to apply the expression on.<br/>
          <br/>
            <i>Enum</i>: logs, spans, unspecified<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>expression</b></td>
        <td>string</td>
        <td>
          Expression to run.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Scope.status
<sup><sup>[↩ Parent](#scope)</sup></sup>



ScopeStatus defines the observed state of Coralogix Scope.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#scopestatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Scope.status.conditions[index]
<sup><sup>[↩ Parent](#scopestatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## TCOLogsPolicies
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






TCOLogsPolicies is the Schema for the tcologspolicies API.
NOTE: This resource performs an atomic overwrite of all existing TCO logs policies
in the backend. Any existing policies not defined in this resource will be
removed. Use with caution as this operation is destructive.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>TCOLogsPolicies</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#tcologspoliciesspec">spec</a></b></td>
        <td>object</td>
        <td>
          TCOLogsPoliciesSpec defines the desired state of Coralogix TCO logs policies.
See also https://coralogix.com/docs/tco-optimizer-api

Added in v0.4.0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#tcologspoliciesstatus">status</a></b></td>
        <td>object</td>
        <td>
          TCOLogsPoliciesStatus defines the observed state of TCOLogsPolicies.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### TCOLogsPolicies.spec
<sup><sup>[↩ Parent](#tcologspolicies)</sup></sup>



TCOLogsPoliciesSpec defines the desired state of Coralogix TCO logs policies.
See also https://coralogix.com/docs/tco-optimizer-api

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#tcologspoliciesspecpoliciesindex">policies</a></b></td>
        <td>[]object</td>
        <td>
          Coralogix TCO-Policies-List.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOLogsPolicies.spec.policies[index]
<sup><sup>[↩ Parent](#tcologspoliciesspec)</sup></sup>



A TCO policy for logs.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the policy.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>priority</b></td>
        <td>enum</td>
        <td>
          The policy priority.<br/>
          <br/>
            <i>Enum</i>: block, high, medium, low<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>severities</b></td>
        <td>[]enum</td>
        <td>
          The severities to apply the policy on.<br/>
          <br/>
            <i>Enum</i>: info, warning, critical, error, debug, verbose<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#tcologspoliciesspecpoliciesindexapplications">applications</a></b></td>
        <td>object</td>
        <td>
          The applications to apply the policy on. Applies the policy on all the applications by default.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#tcologspoliciesspecpoliciesindexarchiveretention">archiveRetention</a></b></td>
        <td>object</td>
        <td>
          Matches the specified retention.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description of the policy.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#tcologspoliciesspecpoliciesindexsubsystems">subsystems</a></b></td>
        <td>object</td>
        <td>
          The subsystems to apply the policy on. Applies the policy on all the subsystems by default.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### TCOLogsPolicies.spec.policies[index].applications
<sup><sup>[↩ Parent](#tcologspoliciesspecpoliciesindex)</sup></sup>



The applications to apply the policy on. Applies the policy on all the applications by default.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>names</b></td>
        <td>[]string</td>
        <td>
          Names to match.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ruleType</b></td>
        <td>enum</td>
        <td>
          Type of matching for the name.<br/>
          <br/>
            <i>Enum</i>: is, is_not, start_with, includes<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOLogsPolicies.spec.policies[index].archiveRetention
<sup><sup>[↩ Parent](#tcologspoliciesspecpoliciesindex)</sup></sup>



Matches the specified retention.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#tcologspoliciesspecpoliciesindexarchiveretentionbackendref">backendRef</a></b></td>
        <td>object</td>
        <td>
          Reference to the retention policy<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOLogsPolicies.spec.policies[index].archiveRetention.backendRef
<sup><sup>[↩ Parent](#tcologspoliciesspecpoliciesindexarchiveretention)</sup></sup>



Reference to the retention policy

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the policy.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOLogsPolicies.spec.policies[index].subsystems
<sup><sup>[↩ Parent](#tcologspoliciesspecpoliciesindex)</sup></sup>



The subsystems to apply the policy on. Applies the policy on all the subsystems by default.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>names</b></td>
        <td>[]string</td>
        <td>
          Names to match.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ruleType</b></td>
        <td>enum</td>
        <td>
          Type of matching for the name.<br/>
          <br/>
            <i>Enum</i>: is, is_not, start_with, includes<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOLogsPolicies.status
<sup><sup>[↩ Parent](#tcologspolicies)</sup></sup>



TCOLogsPoliciesStatus defines the observed state of TCOLogsPolicies.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#tcologspoliciesstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### TCOLogsPolicies.status.conditions[index]
<sup><sup>[↩ Parent](#tcologspoliciesstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## TCOTracesPolicies
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






TCOTracesPolicies is the Schema for the tcotracespolicies API.
NOTE: This resource performs an atomic overwrite of all existing TCO traces policies
in the backend. Any existing policies not defined in this resource will be
removed. Use with caution as this operation is destructive.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>TCOTracesPolicies</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#tcotracespoliciesspec">spec</a></b></td>
        <td>object</td>
        <td>
          TCOTracesPoliciesSpec defines the desired state of Coralogix TCO policies for traces.
See also https://coralogix.com/docs/tco-optimizer-api

Added in v0.4.0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#tcotracespoliciesstatus">status</a></b></td>
        <td>object</td>
        <td>
          TCOTracesPoliciesStatus defines the observed state of TCOTracesPolicies.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec
<sup><sup>[↩ Parent](#tcotracespolicies)</sup></sup>



TCOTracesPoliciesSpec defines the desired state of Coralogix TCO policies for traces.
See also https://coralogix.com/docs/tco-optimizer-api

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#tcotracespoliciesspecpoliciesindex">policies</a></b></td>
        <td>[]object</td>
        <td>
          Coralogix TCO-Policies-List.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec.policies[index]
<sup><sup>[↩ Parent](#tcotracespoliciesspec)</sup></sup>



Coralogix TCO policy for traces.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the policy.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>priority</b></td>
        <td>enum</td>
        <td>
          The policy priority.<br/>
          <br/>
            <i>Enum</i>: block, high, medium, low<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#tcotracespoliciesspecpoliciesindexactions">actions</a></b></td>
        <td>object</td>
        <td>
          The actions to apply the policy on. Applies the policy on all the actions by default.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#tcotracespoliciesspecpoliciesindexapplications">applications</a></b></td>
        <td>object</td>
        <td>
          The applications to apply the policy on. Applies the policy on all the applications by default.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#tcotracespoliciesspecpoliciesindexarchiveretention">archiveRetention</a></b></td>
        <td>object</td>
        <td>
          Matches the specified retention.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description of the policy.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#tcotracespoliciesspecpoliciesindexservices">services</a></b></td>
        <td>object</td>
        <td>
          The services to apply the policy on. Applies the policy on all the services by default.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#tcotracespoliciesspecpoliciesindexsubsystems">subsystems</a></b></td>
        <td>object</td>
        <td>
          The subsystems to apply the policy on. Applies the policy on all the subsystems by default.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#tcotracespoliciesspecpoliciesindextagsindex">tags</a></b></td>
        <td>[]object</td>
        <td>
          The tags to apply the policy on. Applies the policy on all the tags by default.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec.policies[index].actions
<sup><sup>[↩ Parent](#tcotracespoliciesspecpoliciesindex)</sup></sup>



The actions to apply the policy on. Applies the policy on all the actions by default.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>names</b></td>
        <td>[]string</td>
        <td>
          Names to match.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ruleType</b></td>
        <td>enum</td>
        <td>
          Type of matching for the name.<br/>
          <br/>
            <i>Enum</i>: is, is_not, start_with, includes<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec.policies[index].applications
<sup><sup>[↩ Parent](#tcotracespoliciesspecpoliciesindex)</sup></sup>



The applications to apply the policy on. Applies the policy on all the applications by default.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>names</b></td>
        <td>[]string</td>
        <td>
          Names to match.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ruleType</b></td>
        <td>enum</td>
        <td>
          Type of matching for the name.<br/>
          <br/>
            <i>Enum</i>: is, is_not, start_with, includes<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec.policies[index].archiveRetention
<sup><sup>[↩ Parent](#tcotracespoliciesspecpoliciesindex)</sup></sup>



Matches the specified retention.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#tcotracespoliciesspecpoliciesindexarchiveretentionbackendref">backendRef</a></b></td>
        <td>object</td>
        <td>
          Reference to the retention policy<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec.policies[index].archiveRetention.backendRef
<sup><sup>[↩ Parent](#tcotracespoliciesspecpoliciesindexarchiveretention)</sup></sup>



Reference to the retention policy

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the policy.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec.policies[index].services
<sup><sup>[↩ Parent](#tcotracespoliciesspecpoliciesindex)</sup></sup>



The services to apply the policy on. Applies the policy on all the services by default.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>names</b></td>
        <td>[]string</td>
        <td>
          Names to match.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ruleType</b></td>
        <td>enum</td>
        <td>
          Type of matching for the name.<br/>
          <br/>
            <i>Enum</i>: is, is_not, start_with, includes<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec.policies[index].subsystems
<sup><sup>[↩ Parent](#tcotracespoliciesspecpoliciesindex)</sup></sup>



The subsystems to apply the policy on. Applies the policy on all the subsystems by default.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>names</b></td>
        <td>[]string</td>
        <td>
          Names to match.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ruleType</b></td>
        <td>enum</td>
        <td>
          Type of matching for the name.<br/>
          <br/>
            <i>Enum</i>: is, is_not, start_with, includes<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec.policies[index].tags[index]
<sup><sup>[↩ Parent](#tcotracespoliciesspecpoliciesindex)</sup></sup>



TCO Policy tag matching rule.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Tag names to match.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ruleType</b></td>
        <td>enum</td>
        <td>
          Operator to match with.<br/>
          <br/>
            <i>Enum</i>: is, is_not, start_with, includes<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          Values to match for<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.status
<sup><sup>[↩ Parent](#tcotracespolicies)</sup></sup>



TCOTracesPoliciesStatus defines the observed state of TCOTracesPolicies.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#tcotracespoliciesstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.status.conditions[index]
<sup><sup>[↩ Parent](#tcotracespoliciesstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## ViewFolder
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






ViewFolder is the Schema for the viewfolders API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>ViewFolder</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#viewfolderspec">spec</a></b></td>
        <td>object</td>
        <td>
          ViewFolderSpec defines the desired state of folder for views.
See also https://coralogix.com/docs/user-guides/monitoring-and-insights/explore-screen/custom-views/

Added in v0.4.0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#viewfolderstatus">status</a></b></td>
        <td>object</td>
        <td>
          ViewFolderStatus defines the observed state of ViewFolder.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ViewFolder.spec
<sup><sup>[↩ Parent](#viewfolder)</sup></sup>



ViewFolderSpec defines the desired state of folder for views.
See also https://coralogix.com/docs/user-guides/monitoring-and-insights/explore-screen/custom-views/

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the view folder<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### ViewFolder.status
<sup><sup>[↩ Parent](#viewfolder)</sup></sup>



ViewFolderStatus defines the observed state of ViewFolder.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#viewfolderstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ViewFolder.status.conditions[index]
<sup><sup>[↩ Parent](#viewfolderstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## View
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






View is the Schema for the views API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>coralogix.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>View</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#viewspec">spec</a></b></td>
        <td>object</td>
        <td>
          ViewSpec defines the desired state of View.
See also https://coralogix.com/docs/user-guides/monitoring-and-insights/explore-screen/custom-views/

Added in v0.4.0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#viewstatus">status</a></b></td>
        <td>object</td>
        <td>
          ViewStatus defines the observed state of View.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### View.spec
<sup><sup>[↩ Parent](#view)</sup></sup>



ViewSpec defines the desired state of View.
See also https://coralogix.com/docs/user-guides/monitoring-and-insights/explore-screen/custom-views/

Added in v0.4.0

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#viewspecfilters">filters</a></b></td>
        <td>object</td>
        <td>
          Filters is the filters for the view.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the view.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#viewspectimeselection">timeSelection</a></b></td>
        <td>object</td>
        <td>
          TimeSelection is the time selection for the view. Exactly one of quickSelection or customSelection must be set.<br/>
          <br/>
            <i>Validations</i>:<li>has(self.quickSelection) != has(self.customSelection): Exactly one of quickSelection or customSelection must be set</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#viewspecfolder">folder</a></b></td>
        <td>object</td>
        <td>
          Folder is the folder to which the view belongs.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#viewspecsearchquery">searchQuery</a></b></td>
        <td>object</td>
        <td>
          SearchQuery is the search query for the view.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### View.spec.filters
<sup><sup>[↩ Parent](#viewspec)</sup></sup>



Filters is the filters for the view.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#viewspecfiltersfiltersindex">filters</a></b></td>
        <td>[]object</td>
        <td>
          Filters is the list of filters for the view.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### View.spec.filters.filters[index]
<sup><sup>[↩ Parent](#viewspecfilters)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is the name of the filter.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>selectedValues</b></td>
        <td>map[string]boolean</td>
        <td>
          SelectedValues is the selected values for the filter.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### View.spec.timeSelection
<sup><sup>[↩ Parent](#viewspec)</sup></sup>



TimeSelection is the time selection for the view. Exactly one of quickSelection or customSelection must be set.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#viewspectimeselectioncustomselection">customSelection</a></b></td>
        <td>object</td>
        <td>
          CustomSelection is the custom selection for the view.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#viewspectimeselectionquickselection">quickSelection</a></b></td>
        <td>object</td>
        <td>
          QuickSelection is the quick selection for the view.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### View.spec.timeSelection.customSelection
<sup><sup>[↩ Parent](#viewspectimeselection)</sup></sup>



CustomSelection is the custom selection for the view.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fromTime</b></td>
        <td>string</td>
        <td>
          FromTime is the start time for the custom selection.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>toTime</b></td>
        <td>string</td>
        <td>
          ToTime is the end time for the custom selection.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### View.spec.timeSelection.quickSelection
<sup><sup>[↩ Parent](#viewspectimeselection)</sup></sup>



QuickSelection is the quick selection for the view.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>seconds</b></td>
        <td>integer</td>
        <td>
          Seconds is the number of seconds for the quick selection.<br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### View.spec.folder
<sup><sup>[↩ Parent](#viewspec)</sup></sup>



Folder is the folder to which the view belongs.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#viewspecfolderresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          ViewFolder custom resource name and namespace. If namespace is not set, the View namespace will be used.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### View.spec.folder.resourceRef
<sup><sup>[↩ Parent](#viewspecfolder)</sup></sup>



ViewFolder custom resource name and namespace. If namespace is not set, the View namespace will be used.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name of the resource (not id).<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          Kubernetes namespace.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### View.spec.searchQuery
<sup><sup>[↩ Parent](#viewspec)</sup></sup>



SearchQuery is the search query for the view.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>query</b></td>
        <td>string</td>
        <td>
          Query is the search query.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### View.status
<sup><sup>[↩ Parent](#view)</sup></sup>



ViewStatus defines the observed state of View.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#viewstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### View.status.conditions[index]
<sup><sup>[↩ Parent](#viewstatus)</sup></sup>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>
          lastTransitionTime is the last time the condition transitioned from one status to another.
This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.<br/>
          <br/>
            <i>Format</i>: date-time<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>
          message is a human readable message indicating details about the transition.
This may be an empty string.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>
          reason contains a programmatic identifier indicating the reason for the condition's last transition.
Producers of specific condition types may define expected values and meanings for this field,
and whether the values are considered a guaranteed API.
The value should be a CamelCase string.
This field may not be empty.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          status of the condition, one of True, False, Unknown.<br/>
          <br/>
            <i>Enum</i>: True, False, Unknown<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          type of condition in CamelCase or in foo.example.com/CamelCase.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>
          observedGeneration represents the .metadata.generation that the condition was set based upon.
For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
with respect to the current state of the instance.<br/>
          <br/>
            <i>Format</i>: int64<br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

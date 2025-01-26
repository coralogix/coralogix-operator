# API Reference

Packages:

- [coralogix.com/v1alpha1](#coralogixcomv1alpha1)

# coralogix.com/v1alpha1

Resource Types:

- [Alert](#alert)

- [ApiKey](#apikey)

- [CustomRole](#customrole)

- [Group](#group)

- [OutboundWebhook](#outboundwebhook)

- [RecordingRuleGroupSet](#recordingrulegroupset)

- [RuleGroup](#rulegroup)

- [Scope](#scope)

- [TCOLogsPolicies](#tcologspolicies)

- [TCOTracesPolicies](#tcotracespolicies)




## Alert
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






Alert is the Schema for the alerts API

<table>
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
          AlertSpec defines the desired state of Alert<br/>
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



AlertSpec defines the desired state of Alert

<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: Info, Warning, Critical, Error<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>active</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecexpirationdate">expirationDate</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>labels</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupsindex">notificationGroups</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>payloadFilters</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecscheduling">scheduling</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecshowininsight">showInInsight</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType
<sup><sup>[↩ Parent](#alertspec)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetric">metric</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypenewvalue">newValue</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttyperatio">ratio</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypestandard">standard</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetimerelative">timeRelative</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracing">tracing</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypeuniquecount">uniqueCount</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
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





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypeflowstagesindexgroupsindex">groups</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypeflowstagesindextimewindow">timeWindow</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index].groups[index]
<sup><sup>[↩ Parent](#alertspecalerttypeflowstagesindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypeflowstagesindexgroupsindexinnerflowalerts">innerFlowAlerts</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>nextOperator</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: And, Or<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index].groups[index].innerFlowAlerts
<sup><sup>[↩ Parent](#alertspecalerttypeflowstagesindexgroupsindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypeflowstagesindexgroupsindexinnerflowalertsalertsindex">alerts</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: And, Or<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index].groups[index].innerFlowAlerts.alerts[index]
<sup><sup>[↩ Parent](#alertspecalerttypeflowstagesindexgroupsindexinnerflowalerts)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>not</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>userAlertId</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index].timeWindow
<sup><sup>[↩ Parent](#alertspecalerttypeflowstagesindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>hours</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>minutes</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>seconds</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metric
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypemetriclucene">lucene</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricpromql">promql</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metric.lucene
<sup><sup>[↩ Parent](#alertspecalerttypemetric)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypemetricluceneconditions">conditions</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>searchQuery</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metric.lucene.conditions
<sup><sup>[↩ Parent](#alertspecalerttypemetriclucene)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alertWhen</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: More, Less<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>arithmeticOperator</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: Avg, Min, Max, Sum, Count, Percentile<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>metricField</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>threshold</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>timeWindow</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: Minute, FiveMinutes, TenMinutes, FifteenMinutes, TwentyMinutes, ThirtyMinutes, Hour, TwoHours, FourHours, SixHours, TwelveHours, TwentyFourHours<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>arithmeticOperatorModifier</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>groupBy</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricluceneconditionsmanageundetectedvalues">manageUndetectedValues</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>minNonNullValuesPercentage</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replaceMissingValueWithZero</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sampleThresholdPercentage</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metric.lucene.conditions.manageUndetectedValues
<sup><sup>[↩ Parent](#alertspecalerttypemetricluceneconditions)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>autoRetireRatio</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: Never, FiveMinutes, TenMinutes, Hour, TwoHours, SixHours, TwelveHours, TwentyFourHours<br/>
            <i>Default</i>: Never<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enableTriggeringOnUndetectedValues</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metric.promql
<sup><sup>[↩ Parent](#alertspecalerttypemetric)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypemetricpromqlconditions">conditions</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>searchQuery</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metric.promql.conditions
<sup><sup>[↩ Parent](#alertspecalerttypemetricpromql)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alertWhen</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: More, Less, MoreThanUsual<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>threshold</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>timeWindow</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: Minute, FiveMinutes, TenMinutes, FifteenMinutes, TwentyMinutes, ThirtyMinutes, Hour, TwoHours, FourHours, SixHours, TwelveHours, TwentyFourHours<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricpromqlconditionsmanageundetectedvalues">manageUndetectedValues</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>minNonNullValuesPercentage</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replaceMissingValueWithZero</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>sampleThresholdPercentage</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Minimum</i>: 0<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metric.promql.conditions.manageUndetectedValues
<sup><sup>[↩ Parent](#alertspecalerttypemetricpromqlconditions)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>autoRetireRatio</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: Never, FiveMinutes, TenMinutes, Hour, TwoHours, SixHours, TwelveHours, TwentyFourHours<br/>
            <i>Default</i>: Never<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enableTriggeringOnUndetectedValues</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.newValue
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypenewvalueconditions">conditions</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypenewvaluefilters">filters</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.newValue.conditions
<sup><sup>[↩ Parent](#alertspecalerttypenewvalue)</sup></sup>





<table>
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
        <td><b>timeWindow</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: TwelveHours, TwentyFourHours, FortyEightHours, SeventTwoHours, Week, Month, TwoMonths, ThreeMonths<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.newValue.filters
<sup><sup>[↩ Parent](#alertspecalerttypenewvalue)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alias</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>applications</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>categories</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>classes</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>computers</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ips</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>methods</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>searchQuery</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severities</b></td>
        <td>[]enum</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subsystems</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.ratio
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttyperatioconditions">conditions</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttyperatioq1filters">q1Filters</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttyperatioq2filters">q2Filters</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.ratio.conditions
<sup><sup>[↩ Parent](#alertspecalerttyperatio)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alertWhen</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: More, Less<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ratio</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>timeWindow</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: FiveMinutes, TenMinutes, FifteenMinutes, TwentyMinutes, ThirtyMinutes, Hour, TwoHours, FourHours, SixHours, TwelveHours, TwentyFourHours, ThirtySixHours<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>groupBy</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>groupByFor</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: Q1, Q2, Both<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ignoreInfinity</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttyperatioconditionsmanageundetectedvalues">manageUndetectedValues</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.ratio.conditions.manageUndetectedValues
<sup><sup>[↩ Parent](#alertspecalerttyperatioconditions)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>autoRetireRatio</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: Never, FiveMinutes, TenMinutes, Hour, TwoHours, SixHours, TwelveHours, TwentyFourHours<br/>
            <i>Default</i>: Never<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enableTriggeringOnUndetectedValues</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.ratio.q1Filters
<sup><sup>[↩ Parent](#alertspecalerttyperatio)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alias</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>applications</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>categories</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>classes</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>computers</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ips</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>methods</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>searchQuery</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severities</b></td>
        <td>[]enum</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subsystems</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.ratio.q2Filters
<sup><sup>[↩ Parent](#alertspecalerttyperatio)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alias</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>applications</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>searchQuery</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severities</b></td>
        <td>[]enum</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subsystems</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.standard
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypestandardconditions">conditions</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypestandardfilters">filters</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.standard.conditions
<sup><sup>[↩ Parent](#alertspecalerttypestandard)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alertWhen</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: More, Less, Immediately, MoreThanUsual<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>groupBy</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypestandardconditionsmanageundetectedvalues">manageUndetectedValues</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>threshold</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>timeWindow</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: FiveMinutes, TenMinutes, FifteenMinutes, TwentyMinutes, ThirtyMinutes, Hour, TwoHours, FourHours, SixHours, TwelveHours, TwentyFourHours, ThirtySixHours<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.standard.conditions.manageUndetectedValues
<sup><sup>[↩ Parent](#alertspecalerttypestandardconditions)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>autoRetireRatio</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: Never, FiveMinutes, TenMinutes, Hour, TwoHours, SixHours, TwelveHours, TwentyFourHours<br/>
            <i>Default</i>: Never<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enableTriggeringOnUndetectedValues</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.standard.filters
<sup><sup>[↩ Parent](#alertspecalerttypestandard)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alias</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>applications</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>categories</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>classes</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>computers</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ips</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>methods</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>searchQuery</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severities</b></td>
        <td>[]enum</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subsystems</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.timeRelative
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypetimerelativeconditions">conditions</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetimerelativefilters">filters</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.timeRelative.conditions
<sup><sup>[↩ Parent](#alertspecalerttypetimerelative)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alertWhen</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: More, Less<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>threshold</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>timeWindow</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: PreviousHour, SameHourYesterday, SameHourLastWeek, Yesterday, SameDayLastWeek, SameDayLastMonth<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>groupBy</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ignoreInfinity</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetimerelativeconditionsmanageundetectedvalues">manageUndetectedValues</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.timeRelative.conditions.manageUndetectedValues
<sup><sup>[↩ Parent](#alertspecalerttypetimerelativeconditions)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>autoRetireRatio</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: Never, FiveMinutes, TenMinutes, Hour, TwoHours, SixHours, TwelveHours, TwentyFourHours<br/>
            <i>Default</i>: Never<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>enableTriggeringOnUndetectedValues</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.timeRelative.filters
<sup><sup>[↩ Parent](#alertspecalerttypetimerelative)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alias</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>applications</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>categories</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>classes</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>computers</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ips</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>methods</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>searchQuery</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severities</b></td>
        <td>[]enum</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subsystems</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracing
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypetracingconditions">conditions</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingfilters">filters</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracing.conditions
<sup><sup>[↩ Parent](#alertspecalerttypetracing)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alertWhen</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: More, Immediately<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>groupBy</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>threshold</b></td>
        <td>integer</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>timeWindow</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: FiveMinutes, TenMinutes, FifteenMinutes, TwentyMinutes, ThirtyMinutes, Hour, TwoHours, FourHours, SixHours, TwelveHours, TwentyFourHours, ThirtySixHours<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracing.filters
<sup><sup>[↩ Parent](#alertspecalerttypetracing)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>applications</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>latencyThresholdMilliseconds</b></td>
        <td>int or string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>services</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subsystems</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingfilterstagfiltersindex">tagFilters</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracing.filters.tagFilters[index]
<sup><sup>[↩ Parent](#alertspecalerttypetracingfilters)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>field</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.uniqueCount
<sup><sup>[↩ Parent](#alertspecalerttype)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypeuniquecountconditions">conditions</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypeuniquecountfilters">filters</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.uniqueCount.conditions
<sup><sup>[↩ Parent](#alertspecalerttypeuniquecount)</sup></sup>





<table>
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
        <td><b>maxUniqueValues</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>timeWindow</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: Minute, FiveMinutes, TenMinutes, FifteenMinutes, TwentyMinutes, ThirtyMinutes, Hour, TwoHours, FourHours, SixHours, TwelveHours, TwentyFourHours, ThirtySixHours<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>groupBy</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>maxUniqueValuesForGroupBy</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.uniqueCount.filters
<sup><sup>[↩ Parent](#alertspecalerttypeuniquecount)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>alias</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>applications</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>categories</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>classes</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>computers</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>ips</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>methods</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>searchQuery</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severities</b></td>
        <td>[]enum</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subsystems</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.expirationDate
<sup><sup>[↩ Parent](#alertspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>day</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Minimum</i>: 1<br/>
            <i>Maximum</i>: 31<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>month</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Minimum</i>: 1<br/>
            <i>Maximum</i>: 12<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>year</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Minimum</i>: 1<br/>
            <i>Maximum</i>: 9999<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroups[index]
<sup><sup>[↩ Parent](#alertspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>groupByFields</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupsindexnotificationsindex">notifications</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroups[index].notifications[index]
<sup><sup>[↩ Parent](#alertspecnotificationgroupsindex)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>emailRecipients</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>integrationName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>notifyOn</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: TriggeredOnly, TriggeredAndResolved<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>retriggeringPeriodMinutes</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.scheduling
<sup><sup>[↩ Parent](#alertspec)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>daysEnabled</b></td>
        <td>[]enum</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>endTime</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>startTime</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>timeZone</b></td>
        <td>string</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: UTC+00<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.showInInsight
<sup><sup>[↩ Parent](#alertspec)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: TriggeredOnly, TriggeredAndResolved<br/>
            <i>Default</i>: TriggeredOnly<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>retriggeringPeriodMinutes</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Minimum</i>: 1<br/>
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
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
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
          ApiKeySpec defines the desired state of ApiKey.<br/>
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



ApiKeySpec defines the desired state of ApiKey.

<table>
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
        <td><b><a href="#apikeyspecowner">owner</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>active</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>permissions</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>presets</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### ApiKey.spec.owner
<sup><sup>[↩ Parent](#apikeyspec)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>userId</b></td>
        <td>string</td>
        <td>
          <br/>
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
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
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
          CustomRoleSpec defines the desired state of CustomRole.<br/>
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



CustomRoleSpec defines the desired state of CustomRole.

<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>parentRoleName</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>permissions</b></td>
        <td>[]string</td>
        <td>
          <br/>
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
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
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
          GroupSpec defines the desired state of Group.<br/>
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



GroupSpec defines the desired state of Group.

<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#groupspecmembersindex">members</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#groupspecscope">scope</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Group.spec.customRoles[index]
<sup><sup>[↩ Parent](#groupspec)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Group.spec.customRoles[index].resourceRef
<sup><sup>[↩ Parent](#groupspeccustomrolesindex)</sup></sup>





<table>
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
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Group.spec.members[index]
<sup><sup>[↩ Parent](#groupspec)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Group.spec.scope
<sup><sup>[↩ Parent](#groupspec)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Group.spec.scope.resourceRef
<sup><sup>[↩ Parent](#groupspecscope)</sup></sup>





<table>
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
        <td><b>namespace</b></td>
        <td>string</td>
        <td>
          <br/>
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
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>

## OutboundWebhook
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






OutboundWebhook is the Schema for the outboundwebhooks API

<table>
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
          OutboundWebhookSpec defines the desired state of OutboundWebhook<br/>
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



OutboundWebhookSpec defines the desired state of OutboundWebhook

<table>
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
        <td><b><a href="#outboundwebhookspecoutboundwebhooktype">outboundWebhookType</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType
<sup><sup>[↩ Parent](#outboundwebhookspec)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypedemisto">demisto</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypeemailgroup">emailGroup</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypegenericwebhook">genericWebhook</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypejira">jira</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypemicrosoftteams">microsoftTeams</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypeopsgenie">opsgenie</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypepagerduty">pagerDuty</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypesendlog">sendLog</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypeslack">slack</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.awsEventBridge
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>





<table>
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





<table>
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





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.genericWebhook
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: Unkown, Get, Post, Put<br/>
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
        <td><b>headers</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>payload</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.jira
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>email</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>projectKey</b></td>
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
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.microsoftTeams
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>





<table>
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


### OutboundWebhook.spec.outboundWebhookType.opsgenie
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>





<table>
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





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.sendLog
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>





<table>
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
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.slack
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktype)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#outboundwebhookspecoutboundwebhooktypeslackdigestsindex">digests</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.slack.attachments[index]
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktypeslack)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### OutboundWebhook.spec.outboundWebhookType.slack.digests[index]
<sup><sup>[↩ Parent](#outboundwebhookspecoutboundwebhooktypeslack)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>
          <br/>
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
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>externalId</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## RecordingRuleGroupSet
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






RecordingRuleGroupSet is the Schema for the recordingrulegroupsets API

<table>
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
          RecordingRuleGroupSetSpec defines the desired state of RecordingRuleGroupSet<br/>
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



RecordingRuleGroupSetSpec defines the desired state of RecordingRuleGroupSet

<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RecordingRuleGroupSet.spec.groups[index]
<sup><sup>[↩ Parent](#recordingrulegroupsetspec)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Default</i>: 60<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>limit</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
Important: Run "make" to regenerate code after modifying this file<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#recordingrulegroupsetspecgroupsindexrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RecordingRuleGroupSet.spec.groups[index].rules[index]
<sup><sup>[↩ Parent](#recordingrulegroupsetspecgroupsindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>labels</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>record</b></td>
        <td>string</td>
        <td>
          <br/>
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
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>active</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>applications</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>creator</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>hidden</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>order</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Minimum</i>: 1<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severities</b></td>
        <td>[]enum</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindex">subgroups</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>subsystems</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index]
<sup><sup>[↩ Parent](#rulegroupspec)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>order</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index]
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindex)</sup></sup>





<table>
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
        <td><b>active</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexblock">block</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexextract">extract</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexextracttimestamp">extractTimestamp</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexjsonextract">jsonExtract</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexjsonstringify">jsonStringify</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexparse">parse</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexparsejsonfield">parseJsonField</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexremovefields">removeFields</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#rulegroupspecsubgroupsindexrulesindexreplace">replace</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].block
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>sourceField</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>blockingAllMatchingBlocks</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>keepBlockedLogs</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].extract
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>sourceField</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].extractTimestamp
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: Strftime, JavaSDF, Golang, SecondTS, MilliTS, MicroTS, NanoTS<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>sourceField</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>timeFormat</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].jsonExtract
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: Category, CLASSNAME, METHODNAME, THREADID, SEVERITY<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>jsonKey</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].jsonStringify
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>sourceField</b></td>
        <td>string</td>
        <td>
          <br/>
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





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>regex</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>sourceField</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].parseJsonField
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>keepDestinationField</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>keepSourceField</b></td>
        <td>boolean</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>sourceField</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].removeFields
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### RuleGroup.spec.subgroups[index].rules[index].replace
<sup><sup>[↩ Parent](#rulegroupspecsubgroupsindexrulesindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>regex</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>replacementString</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>sourceField</b></td>
        <td>string</td>
        <td>
          <br/>
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
        <td><b>id</b></td>
        <td>string</td>
        <td>
          INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
Important: Run "make" to regenerate code after modifying this file<br/>
        </td>
        <td>true</td>
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
          ScopeSpec defines the desired state of Scope.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#scopestatus">status</a></b></td>
        <td>object</td>
        <td>
          ScopeStatus defines the observed state of Scope.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Scope.spec
<sup><sup>[↩ Parent](#scope)</sup></sup>



ScopeSpec defines the desired state of Scope.

<table>
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
          <br/>
          <br/>
            <i>Enum</i>: <v1>true, <v1>false<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#scopespecfiltersindex">filters</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Scope.spec.filters[index]
<sup><sup>[↩ Parent](#scopespec)</sup></sup>



ScopeFilter defines a filter for a scope

<table>
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
          <br/>
          <br/>
            <i>Enum</i>: logs, spans, unspecified<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>expression</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Scope.status
<sup><sup>[↩ Parent](#scope)</sup></sup>



ScopeStatus defines the observed state of Scope.

<table>
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
          TCOLogsPoliciesSpec defines the desired state of TCOLogsPolicies.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>object</td>
        <td>
          TCOLogsPoliciesStatus defines the observed state of TCOLogsPolicies.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### TCOLogsPolicies.spec
<sup><sup>[↩ Parent](#tcologspolicies)</sup></sup>



TCOLogsPoliciesSpec defines the desired state of TCOLogsPolicies.

<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOLogsPolicies.spec.policies[index]
<sup><sup>[↩ Parent](#tcologspoliciesspec)</sup></sup>





<table>
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
        <td><b>priority</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: block, high, medium, low<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>severities</b></td>
        <td>[]enum</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#tcologspoliciesspecpoliciesindexapplications">applications</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>archiveRetentionId</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#tcologspoliciesspecpoliciesindexsubsystems">subsystems</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### TCOLogsPolicies.spec.policies[index].applications
<sup><sup>[↩ Parent](#tcologspoliciesspecpoliciesindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ruleType</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: is, is_not, start_with, includes<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOLogsPolicies.spec.policies[index].subsystems
<sup><sup>[↩ Parent](#tcologspoliciesspecpoliciesindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ruleType</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: is, is_not, start_with, includes<br/>
        </td>
        <td>true</td>
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
          TCOTracesPoliciesSpec defines the desired state of TCOTracesPolicies.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>object</td>
        <td>
          TCOTracesPoliciesStatus defines the observed state of TCOTracesPolicies.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec
<sup><sup>[↩ Parent](#tcotracespolicies)</sup></sup>



TCOTracesPoliciesSpec defines the desired state of TCOTracesPolicies.

<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec.policies[index]
<sup><sup>[↩ Parent](#tcotracespoliciesspec)</sup></sup>





<table>
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
        <td><b>priority</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: block, high, medium, low<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>severities</b></td>
        <td>[]enum</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#tcotracespoliciesspecpoliciesindexactions">actions</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#tcotracespoliciesspecpoliciesindexapplications">applications</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>archiveRetentionId</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#tcotracespoliciesspecpoliciesindexservices">services</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#tcotracespoliciesspecpoliciesindexsubsystems">subsystems</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#tcotracespoliciesspecpoliciesindextagsindex">tags</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec.policies[index].actions
<sup><sup>[↩ Parent](#tcotracespoliciesspecpoliciesindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ruleType</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: is, is_not, start_with, includes<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec.policies[index].applications
<sup><sup>[↩ Parent](#tcotracespoliciesspecpoliciesindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ruleType</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: is, is_not, start_with, includes<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec.policies[index].services
<sup><sup>[↩ Parent](#tcotracespoliciesspecpoliciesindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ruleType</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: is, is_not, start_with, includes<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec.policies[index].subsystems
<sup><sup>[↩ Parent](#tcotracespoliciesspecpoliciesindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>ruleType</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: is, is_not, start_with, includes<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec.policies[index].tags[index]
<sup><sup>[↩ Parent](#tcotracespoliciesspecpoliciesindex)</sup></sup>





<table>
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
        <td><b>ruleType</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: is, is_not, start_with, includes<br/>
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

# API Reference

Packages:

- [coralogix.com/v1alpha1](#coralogixcomv1alpha1)
- [coralogix.com/v1beta1](#coralogixcomv1beta1)

# coralogix.com/v1alpha1

Resource Types:

- [Alert](#alert)

- [AlertScheduler](#alertscheduler)

- [ApiKey](#apikey)

- [CustomRole](#customrole)

- [Dashboard](#dashboard)

- [DashboardsFolder](#dashboardsfolder)

- [Group](#group)

- [Integration](#integration)

- [OutboundWebhook](#outboundwebhook)

- [RecordingRuleGroupSet](#recordingrulegroupset)

- [RuleGroup](#rulegroup)

- [Scope](#scope)

- [TCOLogsPolicies](#tcologspolicies)

- [TCOTracesPolicies](#tcotracespolicies)




## Alert
<sup><sup>[↩ Parent](#coralogixcomv1alpha1 )</sup></sup>






Alert is the v1alpha1 version Schema for the alerts API. v1alpha1 Alert is going to be deprecated, consider using v1beta1.Alert instead.

<table>
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
          AlertSpec defines the desired state of Alert.<br/>
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



AlertSpec defines the desired state of Alert.

<table>
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
            <i>Enum</i>: Info, Warning, Critical, Error, Low<br/>
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
            <i>Enum</i>: Minute, FiveMinutes, TenMinutes, FifteenMinutes, TwentyMinutes, ThirtyMinutes, Hour, TwoHours, FourHours, SixHours, TwelveHours, TwentyFourHours, ThirtySixHours<br/>
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
            <i>Enum</i>: More, Less, MoreOrEqual, LessOrEqual, MoreThanUsual, LessThanUsual<br/>
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
            <i>Enum</i>: Minute, FiveMinutes, TenMinutes, FifteenMinutes, TwentyMinutes, ThirtyMinutes, Hour, TwoHours, FourHours, SixHours, TwelveHours, TwentyFourHours, ThirtySixHours<br/>
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
            <i>Enum</i>: TwelveHours, TwentyFourHours, FortyEightHours, SeventyTwoHours, Week, Month, TwoMonths, ThreeMonths<br/>
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
          <br/>
            <i>Enum</i>: Debug, Verbose, Info, Warning, Critical, Error<br/>
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
          <br/>
            <i>Enum</i>: Debug, Verbose, Info, Warning, Critical, Error<br/>
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
          <br/>
            <i>Enum</i>: Debug, Verbose, Info, Warning, Critical, Error<br/>
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
          <br/>
            <i>Enum</i>: Debug, Verbose, Info, Warning, Critical, Error<br/>
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
          <br/>
            <i>Enum</i>: Debug, Verbose, Info, Warning, Critical, Error<br/>
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
          <br/>
            <i>Enum</i>: Debug, Verbose, Info, Warning, Critical, Error<br/>
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
          <br/>
            <i>Enum</i>: Sunday, Monday, Tuesday, Wednesday, Thursday, Friday, Saturday<br/>
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
It is used to suppress or activate alerts based on a schedule.<br/>
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

<table>
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
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertschedulerspecschedulerecurring">recurring</a></b></td>
        <td>object</td>
        <td>
          Recurring schedule. Conflicts with `oneTime`.<br/>
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
          DashboardSpec defines the desired state of Dashboard.<br/>
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

<table>
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
          <br/>
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





<table>
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
          DashboardsFolderSpec defines the desired state of DashboardsFolder.<br/>
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



DashboardsFolderSpec defines the desired state of DashboardsFolder.

<table>
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
          IntegrationSpec defines the desired state of Integration.<br/>
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



IntegrationSpec defines the desired state of Integration.

<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>parameters</b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>version</b></td>
        <td>string</td>
        <td>
          <br/>
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
          <br/>
            <i>Enum</i>: Debug, Verbose, Info, Warning, Error, Critical<br/>
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
          TCOLogsPoliciesSpec defines the desired state of TCOLogsPolicies.<br/>
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
          <br/>
            <i>Enum</i>: info, warning, critical, error, debug, verbose<br/>
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
        <td><b><a href="#tcologspoliciesspecpoliciesindexarchiveretention">archiveRetention</a></b></td>
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


### TCOLogsPolicies.spec.policies[index].archiveRetention
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
        <td><b><a href="#tcologspoliciesspecpoliciesindexarchiveretentionbackendref">backendRef</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOLogsPolicies.spec.policies[index].archiveRetention.backendRef
<sup><sup>[↩ Parent](#tcologspoliciesspecpoliciesindexarchiveretention)</sup></sup>





<table>
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
          TCOTracesPoliciesSpec defines the desired state of TCOTracesPolicies.<br/>
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
        <td><b><a href="#tcotracespoliciesspecpoliciesindexarchiveretention">archiveRetention</a></b></td>
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


### TCOTracesPolicies.spec.policies[index].archiveRetention
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
        <td><b><a href="#tcotracespoliciesspecpoliciesindexarchiveretentionbackendref">backendRef</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### TCOTracesPolicies.spec.policies[index].archiveRetention.backendRef
<sup><sup>[↩ Parent](#tcotracespoliciesspecpoliciesindexarchiveretention)</sup></sup>





<table>
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
        <td><b><a href="#alertspec-1">spec</a></b></td>
        <td>object</td>
        <td>
          AlertSpec defines the desired state of Alert<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertstatus-1">status</a></b></td>
        <td>object</td>
        <td>
          AlertStatus defines the observed state of Alert<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec
<sup><sup>[↩ Parent](#alert-1)</sup></sup>



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
        <td><b><a href="#alertspecalerttype-1">alertType</a></b></td>
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
        <td><b>priority</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: p1, p2, p3, p4, p5<br/>
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
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: true<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>entityLabels</b></td>
        <td>map[string]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>groupByKeys</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecincidentssettings">incidentsSettings</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroup">notificationGroup</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindex">notificationGroupExcess</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
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
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType
<sup><sup>[↩ Parent](#alertspec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#alertspecalerttypeflow-1">flow</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsanomaly">logsAnomaly</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsimmediate">logsImmediate</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsnewvalue">logsNewValue</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsratiothreshold">logsRatioThreshold</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsthreshold">logsThreshold</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogstimerelativethreshold">logsTimeRelativeThreshold</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsuniquecount">logsUniqueCount</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricanomaly">metricAnomaly</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricthreshold">metricThreshold</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingimmediate">tracingImmediate</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingthreshold">tracingThreshold</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow
<sup><sup>[↩ Parent](#alertspecalerttype-1)</sup></sup>





<table>
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
        <td><b><a href="#alertspecalerttypeflowstagesindex-1">stages</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index]
<sup><sup>[↩ Parent](#alertspecalerttypeflow-1)</sup></sup>





<table>
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
          <br/>
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
          <br/>
          <br/>
            <i>Enum</i>: unspecified, upTo<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index].flowStagesType
<sup><sup>[↩ Parent](#alertspecalerttypeflowstagesindex-1)</sup></sup>





<table>
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





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>alertsOp</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: and, or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>nextOp</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: and, or<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index].flowStagesType.groups[index].alertDefs[index]
<sup><sup>[↩ Parent](#alertspecalerttypeflowstagesindexflowstagestypegroupsindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>not</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index].flowStagesType.groups[index].alertDefs[index].alertRef
<sup><sup>[↩ Parent](#alertspecalerttypeflowstagesindexflowstagestypegroupsindexalertdefsindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypeflowstagesindexflowstagestypegroupsindexalertdefsindexalertrefresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index].flowStagesType.groups[index].alertDefs[index].alertRef.backendRef
<sup><sup>[↩ Parent](#alertspecalerttypeflowstagesindexflowstagestypegroupsindexalertdefsindexalertref)</sup></sup>





<table>
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
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.flow.stages[index].flowStagesType.groups[index].alertDefs[index].alertRef.resourceRef
<sup><sup>[↩ Parent](#alertspecalerttypeflowstagesindexflowstagestypegroupsindexalertdefsindexalertref)</sup></sup>





<table>
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


### Alert.spec.alertType.logsAnomaly
<sup><sup>[↩ Parent](#alertspecalerttype-1)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsanomalylogsfilter">logsFilter</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>notificationPayloadFilter</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsanomaly)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypelogsanomalyrulesindex)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Default</i>: 0<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsanomalyrulesindexconditiontimewindow">timeWindow</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly.rules[index].condition.timeWindow
<sup><sup>[↩ Parent](#alertspecalerttypelogsanomalyrulesindexcondition)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: 5m, 10m, 15m, 30m, 1h, 2h, 6h, 12h, 24h, 36h<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly.logsFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsanomaly)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly.logsFilter.simpleFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsanomalylogsfilter)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>luceneQuery</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly.logsFilter.simpleFilter.labelFilters
<sup><sup>[↩ Parent](#alertspecalerttypelogsanomalylogsfiltersimplefilter)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>[]enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: debug, info, warning, error, critical, verbose<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsanomalylogsfiltersimplefilterlabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly.logsFilter.simpleFilter.labelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsanomalylogsfiltersimplefilterlabelfilters)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith<br/>
            <i>Default</i>: or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsAnomaly.logsFilter.simpleFilter.labelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsanomalylogsfiltersimplefilterlabelfilters)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith<br/>
            <i>Default</i>: or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsImmediate
<sup><sup>[↩ Parent](#alertspecalerttype-1)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>notificationPayloadFilter</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsImmediate.logsFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsimmediate)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsImmediate.logsFilter.simpleFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsimmediatelogsfilter)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>luceneQuery</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsImmediate.logsFilter.simpleFilter.labelFilters
<sup><sup>[↩ Parent](#alertspecalerttypelogsimmediatelogsfiltersimplefilter)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>[]enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: debug, info, warning, error, critical, verbose<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsimmediatelogsfiltersimplefilterlabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsImmediate.logsFilter.simpleFilter.labelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsimmediatelogsfiltersimplefilterlabelfilters)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith<br/>
            <i>Default</i>: or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsImmediate.logsFilter.simpleFilter.labelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsimmediatelogsfiltersimplefilterlabelfilters)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith<br/>
            <i>Default</i>: or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue
<sup><sup>[↩ Parent](#alertspecalerttype-1)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsnewvaluerulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>notificationPayloadFilter</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue.logsFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsnewvalue)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue.logsFilter.simpleFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsnewvaluelogsfilter)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>luceneQuery</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue.logsFilter.simpleFilter.labelFilters
<sup><sup>[↩ Parent](#alertspecalerttypelogsnewvaluelogsfiltersimplefilter)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>[]enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: debug, info, warning, error, critical, verbose<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsnewvaluelogsfiltersimplefilterlabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue.logsFilter.simpleFilter.labelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsnewvaluelogsfiltersimplefilterlabelfilters)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith<br/>
            <i>Default</i>: or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue.logsFilter.simpleFilter.labelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsnewvaluelogsfiltersimplefilterlabelfilters)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith<br/>
            <i>Default</i>: or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsnewvalue)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypelogsnewvaluerulesindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsnewvaluerulesindexconditiontimewindow">timeWindow</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsNewValue.rules[index].condition.timeWindow
<sup><sup>[↩ Parent](#alertspecalerttypelogsnewvaluerulesindexcondition)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: 12h, 24h, 48h, 72h, 1w, 1mo, 2mo, 3mo<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold
<sup><sup>[↩ Parent](#alertspecalerttype-1)</sup></sup>





<table>
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
          <br/>
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
          <br/>
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
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.denominator
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothreshold)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.denominator.simpleFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholddenominator)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>luceneQuery</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.denominator.simpleFilter.labelFilters
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholddenominatorsimplefilter)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>[]enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: debug, info, warning, error, critical, verbose<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholddenominatorsimplefilterlabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.denominator.simpleFilter.labelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholddenominatorsimplefilterlabelfilters)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith<br/>
            <i>Default</i>: or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.denominator.simpleFilter.labelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholddenominatorsimplefilterlabelfilters)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith<br/>
            <i>Default</i>: or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.numerator
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothreshold)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.numerator.simpleFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholdnumerator)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>luceneQuery</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.numerator.simpleFilter.labelFilters
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholdnumeratorsimplefilter)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>[]enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: debug, info, warning, error, critical, verbose<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholdnumeratorsimplefilterlabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.numerator.simpleFilter.labelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholdnumeratorsimplefilterlabelfilters)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith<br/>
            <i>Default</i>: or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.numerator.simpleFilter.labelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholdnumeratorsimplefilterlabelfilters)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith<br/>
            <i>Default</i>: or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothreshold)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsratiothresholdrulesindexoverride">override</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholdrulesindex)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: moreThan, lessThan<br/>
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
        <td><b><a href="#alertspecalerttypelogsratiothresholdrulesindexconditiontimewindow">timeWindow</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.rules[index].condition.timeWindow
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholdrulesindexcondition)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: 5m, 10m, 15m, 30m, 1h, 2h, 4h, 6h, 12h, 24h, 36h<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsRatioThreshold.rules[index].override
<sup><sup>[↩ Parent](#alertspecalerttypelogsratiothresholdrulesindex)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: p1, p2, p3, p4, p5<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold
<sup><sup>[↩ Parent](#alertspecalerttype-1)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsthresholdlogsfilter">logsFilter</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>notificationPayloadFilter</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsthresholdundetectedvaluesmanagement">undetectedValuesManagement</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsthreshold)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsthresholdrulesindexoverride">override</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypelogsthresholdrulesindex)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: moreThan, lessThan<br/>
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
        <td><b><a href="#alertspecalerttypelogsthresholdrulesindexconditiontimewindow">timeWindow</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.rules[index].condition.timeWindow
<sup><sup>[↩ Parent](#alertspecalerttypelogsthresholdrulesindexcondition)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: 5m, 10m, 15m, 30m, 1h, 2h, 6h, 12h, 24h, 36h<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.rules[index].override
<sup><sup>[↩ Parent](#alertspecalerttypelogsthresholdrulesindex)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: p1, p2, p3, p4, p5<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.logsFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsthreshold)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.logsFilter.simpleFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsthresholdlogsfilter)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>luceneQuery</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.logsFilter.simpleFilter.labelFilters
<sup><sup>[↩ Parent](#alertspecalerttypelogsthresholdlogsfiltersimplefilter)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>[]enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: debug, info, warning, error, critical, verbose<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsthresholdlogsfiltersimplefilterlabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.logsFilter.simpleFilter.labelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsthresholdlogsfiltersimplefilterlabelfilters)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith<br/>
            <i>Default</i>: or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.logsFilter.simpleFilter.labelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsthresholdlogsfiltersimplefilterlabelfilters)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith<br/>
            <i>Default</i>: or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsThreshold.undetectedValuesManagement
<sup><sup>[↩ Parent](#alertspecalerttypelogsthreshold)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: never, 5m, 10m, 1h, 2h, 6h, 12h, 24h<br/>
            <i>Default</i>: never<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>triggerUndetectedValues</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold
<sup><sup>[↩ Parent](#alertspecalerttype-1)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogstimerelativethresholdlogsfilter">logsFilter</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogstimerelativethresholdrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>notificationPayloadFilter</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogstimerelativethresholdundetectedvaluesmanagement">undetectedValuesManagement</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.logsFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethreshold)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.logsFilter.simpleFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethresholdlogsfilter)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>luceneQuery</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.logsFilter.simpleFilter.labelFilters
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethresholdlogsfiltersimplefilter)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>[]enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: debug, info, warning, error, critical, verbose<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogstimerelativethresholdlogsfiltersimplefilterlabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.logsFilter.simpleFilter.labelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethresholdlogsfiltersimplefilterlabelfilters)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith<br/>
            <i>Default</i>: or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.logsFilter.simpleFilter.labelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethresholdlogsfiltersimplefilterlabelfilters)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith<br/>
            <i>Default</i>: or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethreshold)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogstimerelativethresholdrulesindexoverride">override</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethresholdrulesindex)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: previousHour, sameHourYesterday, sameHourLastWeek, yesterday, sameDayLastWeek, sameDayLastMonth<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>conditionType</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: moreThan, lessThan<br/>
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


### Alert.spec.alertType.logsTimeRelativeThreshold.rules[index].override
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethresholdrulesindex)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: p1, p2, p3, p4, p5<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsTimeRelativeThreshold.undetectedValuesManagement
<sup><sup>[↩ Parent](#alertspecalerttypelogstimerelativethreshold)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: never, 5m, 10m, 1h, 2h, 6h, 12h, 24h<br/>
            <i>Default</i>: never<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>triggerUndetectedValues</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount
<sup><sup>[↩ Parent](#alertspecalerttype-1)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsuniquecountrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
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
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount.logsFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsuniquecount)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount.logsFilter.simpleFilter
<sup><sup>[↩ Parent](#alertspecalerttypelogsuniquecountlogsfilter)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>luceneQuery</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount.logsFilter.simpleFilter.labelFilters
<sup><sup>[↩ Parent](#alertspecalerttypelogsuniquecountlogsfiltersimplefilter)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>severity</b></td>
        <td>[]enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: debug, info, warning, error, critical, verbose<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsuniquecountlogsfiltersimplefilterlabelfilterssubsystemnameindex">subsystemName</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount.logsFilter.simpleFilter.labelFilters.applicationName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsuniquecountlogsfiltersimplefilterlabelfilters)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith<br/>
            <i>Default</i>: or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount.logsFilter.simpleFilter.labelFilters.subsystemName[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsuniquecountlogsfiltersimplefilterlabelfilters)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith<br/>
            <i>Default</i>: or<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypelogsuniquecount)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypelogsuniquecountrulesindex)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Format</i>: int64<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypelogsuniquecountrulesindexconditiontimewindow">timeWindow</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.logsUniqueCount.rules[index].condition.timeWindow
<sup><sup>[↩ Parent](#alertspecalerttypelogsuniquecountrulesindexcondition)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: 1m, 5m, 10m, 15m, 20m, 30m, 1h, 2h, 4h, 6h, 12h, 24h, 36h<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricAnomaly
<sup><sup>[↩ Parent](#alertspecalerttype-1)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricanomalyrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricAnomaly.metricFilter
<sup><sup>[↩ Parent](#alertspecalerttypemetricanomaly)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricAnomaly.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypemetricanomaly)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricAnomaly.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypemetricanomalyrulesindex)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: moreThanUsual, lessThanUsual<br/>
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
        <td><b>minNonNullValuesPct</b></td>
        <td>integer</td>
        <td>
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Maximum</i>: 100<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricanomalyrulesindexconditionofthelast">ofTheLast</a></b></td>
        <td>object</td>
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
      </tr></tbody>
</table>


### Alert.spec.alertType.metricAnomaly.rules[index].condition.ofTheLast
<sup><sup>[↩ Parent](#alertspecalerttypemetricanomalyrulesindexcondition)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: 1m, 5m, 10m, 15m, 20m, 30m, 1h, 2h, 4h, 6h, 12h, 24h, 36h<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricThreshold
<sup><sup>[↩ Parent](#alertspecalerttype-1)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricthresholdmissingvalues">missingValues</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricthresholdrulesindex">rules</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricthresholdundetectedvaluesmanagement">undetectedValuesManagement</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricThreshold.metricFilter
<sup><sup>[↩ Parent](#alertspecalerttypemetricthreshold)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricThreshold.missingValues
<sup><sup>[↩ Parent](#alertspecalerttypemetricthreshold)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
            <i>Maximum</i>: 100<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>replaceWithZero</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricThreshold.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypemetricthreshold)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypemetricthresholdrulesindexoverride">override</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricThreshold.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypemetricthresholdrulesindex)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: moreThan, lessThan<br/>
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
      </tr></tbody>
</table>


### Alert.spec.alertType.metricThreshold.rules[index].condition.ofTheLast
<sup><sup>[↩ Parent](#alertspecalerttypemetricthresholdrulesindexcondition)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: 1m, 5m, 10m, 15m, 20m, 30m, 1h, 2h, 4h, 6h, 12h, 24h, 36h<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricThreshold.rules[index].override
<sup><sup>[↩ Parent](#alertspecalerttypemetricthresholdrulesindex)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: p1, p2, p3, p4, p5<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.metricThreshold.undetectedValuesManagement
<sup><sup>[↩ Parent](#alertspecalerttypemetricthreshold)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: never, 5m, 10m, 1h, 2h, 6h, 12h, 24h<br/>
            <i>Default</i>: never<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>triggerUndetectedValues</b></td>
        <td>boolean</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: false<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingImmediate
<sup><sup>[↩ Parent](#alertspecalerttype-1)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingimmediatetracingfilter">tracingFilter</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingImmediate.tracingFilter
<sup><sup>[↩ Parent](#alertspecalerttypetracingimmediate)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingImmediate.tracingFilter.simple
<sup><sup>[↩ Parent](#alertspecalerttypetracingimmediatetracingfilter)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingImmediate.tracingFilter.simple.tracingLabelFilters
<sup><sup>[↩ Parent](#alertspecalerttypetracingimmediatetracingfiltersimple)</sup></sup>





<table>
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





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith, isNot<br/>
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





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith, isNot<br/>
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





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith, isNot<br/>
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





<table>
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
          <br/>
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





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith, isNot<br/>
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





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith, isNot<br/>
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
<sup><sup>[↩ Parent](#alertspecalerttype-1)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>notificationPayloadFilter</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingthresholdtracingfilter">tracingFilter</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.rules[index]
<sup><sup>[↩ Parent](#alertspecalerttypetracingthreshold)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.rules[index].condition
<sup><sup>[↩ Parent](#alertspecalerttypetracingthresholdrulesindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecalerttypetracingthresholdrulesindexconditiontimewindow">timeWindow</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.rules[index].condition.timeWindow
<sup><sup>[↩ Parent](#alertspecalerttypetracingthresholdrulesindexcondition)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: 5m, 10m, 15m, 20m, 30m, 1h, 2h, 4h, 6h, 12h, 24h, 36h<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.tracingFilter
<sup><sup>[↩ Parent](#alertspecalerttypetracingthreshold)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.tracingFilter.simple
<sup><sup>[↩ Parent](#alertspecalerttypetracingthresholdtracingfilter)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.alertType.tracingThreshold.tracingFilter.simple.tracingLabelFilters
<sup><sup>[↩ Parent](#alertspecalerttypetracingthresholdtracingfiltersimple)</sup></sup>





<table>
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





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith, isNot<br/>
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





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith, isNot<br/>
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





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith, isNot<br/>
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





<table>
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
          <br/>
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





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith, isNot<br/>
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





<table>
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
          <br/>
          <br/>
            <i>Enum</i>: or, includes, endsWith, startsWith, isNot<br/>
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
<sup><sup>[↩ Parent](#alertspec-1)</sup></sup>





<table>
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
            <i>Enum</i>: triggeredOnly, triggeredAndResolved<br/>
            <i>Default</i>: triggeredOnly<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecincidentssettingsretriggeringperiod">retriggeringPeriod</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.incidentsSettings.retriggeringPeriod
<sup><sup>[↩ Parent](#alertspecincidentssettings)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup
<sup><sup>[↩ Parent](#alertspec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>groupByKeys</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupwebhooksindex">webhooks</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.webhooks[index]
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
        <td><b><a href="#alertspecnotificationgroupwebhooksindexintegration">integration</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>notifyOn</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: triggeredOnly, triggeredAndResolved<br/>
            <i>Default</i>: triggeredOnly<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupwebhooksindexretriggeringperiod">retriggeringPeriod</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.webhooks[index].integration
<sup><sup>[↩ Parent](#alertspecnotificationgroupwebhooksindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>recipients</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.webhooks[index].integration.integrationRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupwebhooksindexintegration)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupwebhooksindexintegrationintegrationrefresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.webhooks[index].integration.integrationRef.backendRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupwebhooksindexintegrationintegrationref)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroup.webhooks[index].integration.integrationRef.resourceRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupwebhooksindexintegrationintegrationref)</sup></sup>





<table>
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


### Alert.spec.notificationGroup.webhooks[index].retriggeringPeriod
<sup><sup>[↩ Parent](#alertspecnotificationgroupwebhooksindex)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index]
<sup><sup>[↩ Parent](#alertspec-1)</sup></sup>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>groupByKeys</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexwebhooksindex">webhooks</a></b></td>
        <td>[]object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].webhooks[index]
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
        <td><b><a href="#alertspecnotificationgroupexcessindexwebhooksindexintegration">integration</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>notifyOn</b></td>
        <td>enum</td>
        <td>
          <br/>
          <br/>
            <i>Enum</i>: triggeredOnly, triggeredAndResolved<br/>
            <i>Default</i>: triggeredOnly<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexwebhooksindexretriggeringperiod">retriggeringPeriod</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].webhooks[index].integration
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexwebhooksindex)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>recipients</b></td>
        <td>[]string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].webhooks[index].integration.integrationRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexwebhooksindexintegration)</sup></sup>





<table>
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
          <br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#alertspecnotificationgroupexcessindexwebhooksindexintegrationintegrationrefresourceref">resourceRef</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].webhooks[index].integration.integrationRef.backendRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexwebhooksindexintegrationintegrationref)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.notificationGroupExcess[index].webhooks[index].integration.integrationRef.resourceRef
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexwebhooksindexintegrationintegrationref)</sup></sup>





<table>
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


### Alert.spec.notificationGroupExcess[index].webhooks[index].retriggeringPeriod
<sup><sup>[↩ Parent](#alertspecnotificationgroupexcessindexwebhooksindex)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Format</i>: int32<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.schedule
<sup><sup>[↩ Parent](#alertspec-1)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Default</i>: UTC+00<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#alertspecscheduleactiveon">activeOn</a></b></td>
        <td>object</td>
        <td>
          <br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.spec.schedule.activeOn
<sup><sup>[↩ Parent](#alertspecschedule)</sup></sup>





<table>
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
          <br/>
          <br/>
            <i>Default</i>: 23:59<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>startTime</b></td>
        <td>string</td>
        <td>
          <br/>
          <br/>
            <i>Default</i>: 00:00<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Alert.status
<sup><sup>[↩ Parent](#alert-1)</sup></sup>



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
        <td><b><a href="#alertstatusconditionsindex-1">conditions</a></b></td>
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
<sup><sup>[↩ Parent](#alertstatus-1)</sup></sup>



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

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

import "testing"

func TestDeepEqualNotificationGroupsEquals(t *testing.T) {
	integrationName := "WebhookAlerts"
	notificationGroups := []NotificationGroup{
		{
			GroupByFields: []string{"field1", "field2"},
			Notifications: []Notification{
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					IntegrationName:           &integrationName,
				},
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					EmailRecipients:           []string{"example@coralogix.com"},
				},
			},
		},
		{
			GroupByFields: []string{"field3"},
			Notifications: []Notification{
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					IntegrationName:           &integrationName,
				},
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					EmailRecipients:           []string{"example@coralogix.com"},
				},
			},
		},
	}

	//Changing the order of the notification groups should not affect the result
	actualNotificationGroups := []NotificationGroup{
		{
			GroupByFields: []string{"field3"},
			Notifications: []Notification{
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					EmailRecipients:           []string{"example@coralogix.com"},
				},
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					IntegrationName:           &integrationName,
				},
			},
		},
		{
			GroupByFields: []string{"field1", "field2"},
			Notifications: []Notification{
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					EmailRecipients:           []string{"example@coralogix.com"},
				},
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					IntegrationName:           &integrationName,
				},
			},
		},
	}
	if equal, dif := DeepEqualNotificationGroups(notificationGroups, actualNotificationGroups); !equal {
		t.Error("Expected to be equal but got: ", dif)
	}
}

func TestDeepEqualNotificationGroupsNotEquals(t *testing.T) {
	integrationName := "WebhookAlerts"
	notificationGroups := []NotificationGroup{
		{
			GroupByFields: []string{"field1", "field2"},
			Notifications: []Notification{
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					IntegrationName:           &integrationName,
				},
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					EmailRecipients:           []string{"example@coralogix.com"},
				},
			},
		},
		{
			GroupByFields: []string{"field3"},
			Notifications: []Notification{
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					IntegrationName:           &integrationName,
				},
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					EmailRecipients:           []string{"example@coralogix.com"},
				},
			},
		},
	}

	actualNotificationGroups := []NotificationGroup{
		{
			GroupByFields: []string{"field3"},
			Notifications: []Notification{
				{
					//Changing the RetriggeringPeriodMinutes should affect the result
					RetriggeringPeriodMinutes: 10,
					NotifyOn:                  NotifyOnTriggeredOnly,
					EmailRecipients:           []string{"example@coralogix.com"},
				},
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					IntegrationName:           &integrationName,
				},
			},
		},
		{
			GroupByFields: []string{"field1", "field2"},
			Notifications: []Notification{
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					EmailRecipients:           []string{"example@coralogix.com"},
				},
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					IntegrationName:           &integrationName,
				},
			},
		},
	}

	if equal, _ := DeepEqualNotificationGroups(notificationGroups, actualNotificationGroups); equal {
		t.Error("Expected to be not equal but got")
	}

	actualNotificationGroups = []NotificationGroup{
		{
			//Changing the GroupByFields should affect the result
			GroupByFields: []string{"field3", "field4"},
			Notifications: []Notification{
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					EmailRecipients:           []string{"example@coralogix.com"},
				},
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					IntegrationName:           &integrationName,
				},
			},
		},
		{
			GroupByFields: []string{"field1", "field2"},
			Notifications: []Notification{
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					EmailRecipients:           []string{"example@coralogix.com"},
				},
				{
					RetriggeringPeriodMinutes: 5,
					NotifyOn:                  NotifyOnTriggeredOnly,
					IntegrationName:           &integrationName,
				},
			},
		},
	}

	if equal, _ := DeepEqualNotificationGroups(notificationGroups, actualNotificationGroups); equal {
		t.Error("Expected to be not equal but got")
	}
}

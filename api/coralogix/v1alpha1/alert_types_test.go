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

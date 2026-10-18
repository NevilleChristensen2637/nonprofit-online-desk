package presenceboard

import (
	"reflect"
	"testing"
)

func TestBuildBriefAssignsOnlyOnlinePendingWork(t *testing.T) {
	online := []Member{{ClientID: "staff-ana", DisplayName: "Ana"}, {ClientID: "staff-lee", DisplayName: "Lee"}}
	tests := []struct {
		name        string
		assignments []Assignment
		want        []DeskAction
	}{
		{
			name: "online owners receive their concrete queues",
			assignments: []Assignment{
				{ClientID: "staff-ana", Name: "Ana", Desk: "donor_receipts", Pending: 4},
				{ClientID: "staff-lee", Name: "Lee", Desk: "campaign_reporting", Pending: 1},
			},
			want: []DeskAction{
				{ClientID: "staff-ana", Name: "Ana", Action: "send_receipts", Pending: 4},
				{ClientID: "staff-lee", Name: "Lee", Action: "prepare_report", Pending: 1},
			},
		},
		{
			name: "offline and empty queues stay off the action list",
			assignments: []Assignment{
				{ClientID: "staff-mo", Name: "Mo", Desk: "volunteer_reminders", Pending: 7},
				{ClientID: "staff-ana", Name: "Ana", Desk: "donor_receipts", Pending: 0},
			},
			want: []DeskAction{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			brief, err := BuildBrief("nonprofit-operations", online, test.assignments)
			if err != nil {
				t.Fatalf("BuildBrief() error = %v", err)
			}
			if !reflect.DeepEqual(brief.DeskActions, test.want) {
				t.Fatalf("DeskActions = %#v, want %#v", brief.DeskActions, test.want)
			}
		})
	}
}

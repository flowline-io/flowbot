package dag

import (
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/utils"
	"testing"
)

func TestTopologySort(t *testing.T) {
	type args struct {
		item *model.Dag
	}
	tests := []struct {
		name    string
		args    args
		want    []model.Step
		wantErr bool
	}{
		{
			name: "dag-1",
			args: args{
				item: &model.Dag{
					ID: 1,
					Nodes: []*model.Node{
						{
							Id: "node1",
						},
						{
							Id: "node2",
						},
						{
							Id: "node3",
						},
					},
					Edges: []*model.Edge{
						{
							Source: "node1",
							Target: "node2",
						},
						{
							Source: "node2",
							Target: "node3",
						},
					},
				},
			},
			want: []model.Step{
				{
					NodeID: "node1",
					Depend: []string{},
					State:  model.StepReady,
				},
				{
					NodeID: "node2",
					Depend: []string{"node1"},
					State:  model.StepCreated,
				},
				{
					NodeID: "node3",
					Depend: []string{"node2"},
					State:  model.StepCreated,
				},
			},
			wantErr: false,
		},
		{
			name: "dag-2",
			args: args{
				item: &model.Dag{
					ID: 1,
					Nodes: []*model.Node{
						{
							Id: "node1",
						},
						{
							Id: "node2",
						},
						{
							Id: "node3",
						},
						{
							Id: "node4",
						},
					},
					Edges: []*model.Edge{
						{
							Source: "node1",
							Target: "node2",
						},
						{
							Source: "node1",
							Target: "node3",
						},
						{
							Source: "node2",
							Target: "node4",
						},
						{
							Source: "node3",
							Target: "node4",
						},
					},
				},
			},
			want: []model.Step{
				{
					NodeID: "node1",
					Depend: []string{},
					State:  model.StepReady,
				},
				{
					NodeID: "node2",
					Depend: []string{"node1"},
					State:  model.StepCreated,
				},
				{
					NodeID: "node3",
					Depend: []string{"node1"},
					State:  model.StepCreated,
				},
				{
					NodeID: "node4",
					Depend: []string{"node2", "node3"},
					State:  model.StepCreated,
				},
			},
			wantErr: false,
		},
		{
			name: "dag-3",
			args: args{
				item: &model.Dag{
					ID: 1,
					Nodes: []*model.Node{
						{
							Id: "node1",
						},
						{
							Id: "node2",
						},
						{
							Id: "node3",
						},
						{
							Id: "node4",
						},
						{
							Id: "node5",
						},
					},
					Edges: []*model.Edge{
						{
							Source: "node1",
							Target: "node2",
						},
						{
							Source: "node2",
							Target: "node3",
						},
						{
							Source: "node2",
							Target: "node4",
						},
						{
							Source: "node3",
							Target: "node4",
						},
						{
							Source: "node3",
							Target: "node5",
						},
						{
							Source: "node4",
							Target: "node5",
						},
					},
				},
			},
			want: []model.Step{
				{
					NodeID: "node1",
					Depend: []string{},
					State:  model.StepReady,
				},
				{
					NodeID: "node2",
					Depend: []string{"node1"},
					State:  model.StepCreated,
				},
				{
					NodeID: "node3",
					Depend: []string{"node2"},
					State:  model.StepCreated,
				},
				{
					NodeID: "node4",
					Depend: []string{"node2", "node3"},
					State:  model.StepCreated,
				},
				{
					NodeID: "node5",
					Depend: []string{"node3", "node4"},
					State:  model.StepCreated,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TopologySort(tt.args.item)
			if (err != nil) != tt.wantErr {
				t.Errorf("TopologySort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(tt.want) != len(got) {
				t.Errorf("TopologySort() got = %v, want %v", got, tt.want)
				return
			}
			for i, step := range tt.want {
				if got[i].NodeID != step.NodeID {
					t.Errorf("TopologySort() got = %v, want %v", got[i].NodeID, step.NodeID)
					return
				}
				if !utils.SameStringSlice(got[i].Depend, step.Depend) {
					t.Errorf("TopologySort() got = %v, want %v", got[i].Depend, step.Depend)
					return
				}
				if got[i].State != step.State {
					t.Errorf("TopologySort() got = %v, want %v", got[i].State, step.State)
					return
				}
			}
		})
	}
}

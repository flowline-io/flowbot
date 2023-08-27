package dag

import (
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/internal/types/meta"
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
		want    []meta.Step
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
			want: []meta.Step{
				{
					NodeId:       "node1",
					DependNodeId: []string{},
					State:        model.StepReady,
				},
				{
					NodeId:       "node2",
					DependNodeId: []string{"node1"},
					State:        model.StepCreated,
				},
				{
					NodeId:       "node3",
					DependNodeId: []string{"node2"},
					State:        model.StepCreated,
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
			want: []meta.Step{
				{
					NodeId:       "node1",
					DependNodeId: []string{},
					State:        model.StepReady,
				},
				{
					NodeId:       "node2",
					DependNodeId: []string{"node1"},
					State:        model.StepCreated,
				},
				{
					NodeId:       "node3",
					DependNodeId: []string{"node1"},
					State:        model.StepCreated,
				},
				{
					NodeId:       "node4",
					DependNodeId: []string{"node2", "node3"},
					State:        model.StepCreated,
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
			want: []meta.Step{
				{
					NodeId:       "node1",
					DependNodeId: []string{},
					State:        model.StepReady,
				},
				{
					NodeId:       "node2",
					DependNodeId: []string{"node1"},
					State:        model.StepCreated,
				},
				{
					NodeId:       "node3",
					DependNodeId: []string{"node2"},
					State:        model.StepCreated,
				},
				{
					NodeId:       "node4",
					DependNodeId: []string{"node2", "node3"},
					State:        model.StepCreated,
				},
				{
					NodeId:       "node5",
					DependNodeId: []string{"node3", "node4"},
					State:        model.StepCreated,
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
				if got[i].NodeId != step.NodeId {
					t.Errorf("TopologySort() got = %v, want %v", got[i].NodeId, step.NodeId)
					return
				}
				if !utils.SameStringSlice(got[i].DependNodeId, step.DependNodeId) {
					t.Errorf("TopologySort() got = %v, want %v", got[i].NodeId, step.NodeId)
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

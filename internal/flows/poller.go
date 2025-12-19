package flows

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/store/model"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/trigger"
)

// Poller periodically runs poll-mode trigger rules and emits flow executions.
//
// It uses flow node Variables as persisted state.
type Poller struct {
	store store.Adapter
	queue *QueueManager
	reg   RuleRegistry
}

func NewPoller(storeAdapter store.Adapter, queue *QueueManager, reg RuleRegistry) *Poller {
	if reg == nil {
		reg = NewChatbotRuleRegistry()
	}
	return &Poller{store: storeAdapter, queue: queue, reg: reg}
}

func (p *Poller) Start(ctx context.Context) {
	if p == nil {
		return
	}
	go p.loop(ctx)
}

func (p *Poller) loop(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.tick(ctx)
		}
	}
}

func (p *Poller) tick(ctx context.Context) {
	// Scan all flows. Store adapter returns all rows when uid/topic are empty.
	flows, err := p.store.GetFlows("", "")
	if err != nil {
		flog.Error(fmt.Errorf("poller: failed to list flows: %w", err))
		return
	}
	for _, f := range flows {
		if f == nil || !f.Enabled {
			continue
		}
		nodes, err := p.store.GetFlowNodes(f.ID)
		if err != nil {
			continue
		}
		for _, n := range nodes {
			if n == nil || n.Type != model.NodeTypeTrigger {
				continue
			}
			r, err := p.reg.FindTrigger(n.Bot, n.RuleID)
			if err != nil {
				continue
			}
			if r.Mode != trigger.ModePoll || r.Poll == nil {
				continue
			}

			params := jsonToKV(n.Parameters)
			state := jsonToKV(n.Variables)

			interval := int64(60)
			if v, ok := params.Int64("interval_seconds"); ok && v > 0 {
				interval = v
			}

			nextAt := int64(0)
			if v, ok := state.Int64("_poll_next_at"); ok {
				nextAt = v
			}
			now := time.Now().Unix()
			if nextAt > now {
				continue
			}

			// Poll.
			ctx2 := types.Context{}
			ctx2.SetTimeout(2 * time.Minute)
			ctx2.AsUser = types.Uid(f.UID)
			ctx2.Topic = f.Topic

			res, err := r.Poll(ctx2, params, state)
			if err != nil {
				flog.Error(fmt.Errorf("poller: %d %s/%s: %w", f.ID, n.Bot, n.RuleID, err))
				state["_poll_next_at"] = time.Now().Add(time.Duration(interval) * time.Second).Unix()
				n.Variables = model.JSON(state)
				_ = p.store.UpdateFlowNode(n)
				continue
			}

			// Persist state.
			if res.State == nil {
				res.State = state
			}
			res.State["_poll_next_at"] = time.Now().Add(time.Duration(interval) * time.Second).Unix()
			n.Variables = model.JSON(res.State)
			_ = p.store.UpdateFlowNode(n)

			// Emit executions.
			triggerType := fmt.Sprintf("%s|%s", n.Bot, n.RuleID)
			for _, ev := range res.Events {
				if ev == nil {
					continue
				}
				triggerID := ""
				if id, ok := ev.String("id"); ok {
					triggerID = id
				}
				if p.queue != nil {
					_, err := p.queue.EnqueueFlowExecution(ctx, f.ID, triggerType, triggerID, ev)
					if err != nil {
						flog.Error(fmt.Errorf("poller: enqueue failed: %w", err))
					}
				}
			}
		}
	}
}

func jsonToKV(j model.JSON) types.KV {
	if j == nil {
		return make(types.KV)
	}
	b, err := sonic.Marshal(j)
	if err != nil {
		return make(types.KV)
	}
	out := make(types.KV)
	_ = sonic.Unmarshal(b, &out)
	return out
}

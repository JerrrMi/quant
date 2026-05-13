package ws

import (
	"sync"

	"github.com/JerrrMi/quant/internal/domain/command"
	"github.com/JerrrMi/quant/internal/domain/report"
)

// CommandDedupeKey 生成指令幂等键：优先 idempotency_key，其次 command_id。
func CommandDedupeKey(cmd command.TradeCommand) string {
	if cmd.IdempotencyKey != "" {
		return string(cmd.IdempotencyKey)
	}
	return cmd.CommandID
}

// CommandDedup 进程内骨架：同一 natural command 重连重放时不二次「适用」。
type CommandDedup struct {
	mu   sync.Mutex
	seen map[string]struct{}
}

// NewCommandDedup 构造空表。
func NewCommandDedup() *CommandDedup {
	return &CommandDedup{seen: map[string]struct{}{}}
}

// FirstApply 若是首次见到该幂等键返回 true；重复返回 false（不应再次触发执行管线）。
func (d *CommandDedup) FirstApply(cmd command.TradeCommand) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	k := CommandDedupeKey(cmd)
	if k == "" {
		return true
	}
	if _, ok := d.seen[k]; ok {
		return false
	}
	d.seen[k] = struct{}{}
	return true
}

// DropSeen 测试或会话重置时使用。
func (d *CommandDedup) DropSeen(cmd command.TradeCommand) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.seen, CommandDedupeKey(cmd))
}

// SaasOutboundNeedsReplay 判断 SaaS 侧缓冲帧是否在 Agent last_seen_saas_seq 之后需要投递。
func SaasOutboundNeedsReplay(lastSeenSaasSeq, envelopeSeq int64) bool {
	return envelopeSeq > lastSeenSaasSeq
}

// MergeFillSnapshots 合并 fills：按 FillID 去重，先出现的保留（重放不污染）。
func MergeFillSnapshots(existing []report.FillRecord, incoming []report.FillRecord) []report.FillRecord {
	index := map[string]struct{}{}
	out := append([]report.FillRecord(nil), existing...)
	for _, f := range existing {
		if f.FillID != "" {
			index[f.FillID] = struct{}{}
		}
	}
	for _, f := range incoming {
		if f.FillID == "" {
			out = append(out, f)
			continue
		}
		if _, ok := index[f.FillID]; ok {
			continue
		}
		index[f.FillID] = struct{}{}
		out = append(out, f)
	}
	return out
}

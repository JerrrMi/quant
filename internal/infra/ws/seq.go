package ws

import "sync"

// SeqAllocator 生成本连接出站单调递增 seq（每条出站信封递增 1；与会话绑定）。
type SeqAllocator struct {
	mu      sync.Mutex
	lastSeq int64
}

// Next 返回下一个出站 seq（从 1 开始）。
func (s *SeqAllocator) Next() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastSeq++
	return s.lastSeq
}

// PeekLast 返回已分配的最后一个 seq（未分配时为 0）。
func (s *SeqAllocator) PeekLast() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastSeq
}

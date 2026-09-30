// Package inputbuffer 是连招输入缓冲：按时间窗口留住预输入、按时间顺序单遍匹配连招。
package inputbuffer

import "sort"

// WindowMs 是缓冲窗口。
const WindowMs = 300

// Capacity 是缓冲允许保留的按键数上限。
const Capacity = 64

// storedPress 是带录入序号的按键，序号只用于同一时刻的稳定排序。
type storedPress struct {
	Press
	seq int
}

// Press 是一次按键。
type Press struct {
	Action string
	AtMs   int64
}

// Chain 是一串要按顺序吃掉的招式步骤。
type Chain struct {
	Steps []string
}

// Buffer 是预输入缓冲。
// 缓冲内按键始终按 (AtMs, 录入顺序) 升序保存，因此最旧的按键总在最前面。
type Buffer struct {
	windowMs int64
	presses  []storedPress
	nextSeq  int
	dropped  int
	scanned  int
}

// NewBuffer 建缓冲，windowMs 是缓冲窗口。
func NewBuffer(windowMs int64) *Buffer { return &Buffer{windowMs: windowMs} }

// Press 收一次按键：按时间戳升序（同刻按录入顺序）插入，超过 Capacity 时丢掉最旧的一条。
func (b *Buffer) Press(action string, atMs int64) {
	entry := storedPress{Press: Press{Action: action, AtMs: atMs}, seq: b.nextSeq}
	b.nextSeq++
	position := sort.Search(len(b.presses), func(index int) bool {
		return b.presses[index].AtMs > atMs ||
			(b.presses[index].AtMs == atMs && b.presses[index].seq > entry.seq)
	})
	b.presses = append(b.presses, storedPress{})
	copy(b.presses[position+1:], b.presses[position:])
	b.presses[position] = entry
	if len(b.presses) > Capacity {
		b.presses = b.presses[1:]
		b.dropped++
	}
}

// evictExpired 物理丢弃 nowMs 时刻已过期的按键：nowMs - AtMs > windowMs（严格大于）。
// 缓冲按时间升序保存，过期按键一定是前缀。
func (b *Buffer) evictExpired(nowMs int64) {
	keep := 0
	for keep < len(b.presses) && nowMs-b.presses[keep].AtMs > b.windowMs {
		keep++
	}
	if keep > 0 {
		b.presses = append(b.presses[:0], b.presses[keep:]...)
	}
}

// Buffered 是 nowMs 时刻缓冲里没过期的按键数。
func (b *Buffer) Buffered(nowMs int64) int {
	b.evictExpired(nowMs)
	return len(b.presses)
}

// Dropped 是被容量挤掉的按键数。
func (b *Buffer) Dropped() int { return b.dropped }

// Scanned 是匹配时看过的按键次数（规模观测）。
func (b *Buffer) Scanned() int { return b.scanned }

// Consume 用缓冲里没过期的按键按时间顺序单遍匹配连招，成功后把命中的按键移出缓冲，
// 失败时缓冲一字不动。
func (b *Buffer) Consume(nowMs int64, chain Chain) ([]string, bool) {
	b.evictExpired(nowMs)
	if len(chain.Steps) == 0 || len(b.presses) < len(chain.Steps) {
		return nil, false
	}
	matched := make([]string, 0, len(chain.Steps))
	stepIndex := 0
	cursor := 0
	for cursor < len(b.presses) && stepIndex < len(chain.Steps) {
		b.scanned++
		if b.presses[cursor].Action == chain.Steps[stepIndex] {
			matched = append(matched, b.presses[cursor].Action)
			stepIndex++
		}
		cursor++
	}
	if stepIndex < len(chain.Steps) {
		return nil, false
	}
	rest := b.presses[:0]
	for _, press := range b.presses {
		if len(rest) < cursor-len(chain.Steps) {
			rest = append(rest, press)
			continue
		}
		if press.Action != chain.Steps[len(rest)-(cursor-len(chain.Steps))] {
			rest = append(rest, press)
		}
	}
	b.presses = rest
	return matched, true
}

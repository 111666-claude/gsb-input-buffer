// Package inputbuffer 是连招输入缓冲：按时间窗口留住预输入、按时间顺序单遍匹配连招。
package inputbuffer

import "sort"

// WindowMs 是缓冲窗口。
const WindowMs = 300

// Capacity 是缓冲允许保留的按键数上限。
const Capacity = 64

// Chain 是一串要按顺序吃掉的招式步骤。
type Chain struct {
	Steps []string
}

// Press 是一次按键。
type Press struct {
	Action string
	AtMs   int64
}

// storedPress 带录入序号，同刻按键按录入顺序排。
type storedPress struct {
	Press
	seq int
}

// Buffer 是预输入缓冲。
// 按键始终按 (AtMs, seq) 升序存放，过期按键在查询/匹配时按当前时间清理。
type Buffer struct {
	windowMs int64
	presses  []storedPress
	dropped  int
	scanned  int
	nextSeq  int
}

// NewBuffer 建缓冲，windowMs 是缓冲窗口。
func NewBuffer(windowMs int64) *Buffer { return &Buffer{windowMs: windowMs} }

// Press 收一次按键。超出 Capacity 时丢最旧的一条（AtMs 最早，同刻按录入顺序），
// 丢弃次数累加到 Dropped。
func (b *Buffer) Press(action string, atMs int64) {
	candidate := storedPress{Press: Press{Action: action, AtMs: atMs}, seq: b.nextSeq}
	b.nextSeq++
	pos := sort.Search(len(b.presses), func(i int) bool {
		existing := b.presses[i]
		return existing.AtMs > atMs || (existing.AtMs == atMs && existing.seq > candidate.seq)
	})
	b.presses = append(b.presses, storedPress{})
	copy(b.presses[pos+1:], b.presses[pos:])
	b.presses[pos] = candidate
	if len(b.presses) > Capacity {
		copy(b.presses, b.presses[1:])
		b.presses[len(b.presses)-1] = storedPress{}
		b.presses = b.presses[:len(b.presses)-1]
		b.dropped++
	}
}

// sweepExpired 丢掉 nowMs 时刻已过期的按键：nowMs - AtMs > windowMs。
func (b *Buffer) sweepExpired(nowMs int64) {
	cutoff := nowMs - b.windowMs
	pos := sort.Search(len(b.presses), func(i int) bool { return b.presses[i].AtMs >= cutoff })
	if pos == 0 {
		return
	}
	copy(b.presses, b.presses[pos:])
	kept := len(b.presses) - pos
	for i := kept; i < len(b.presses); i++ {
		b.presses[i] = storedPress{}
	}
	b.presses = b.presses[:kept]
}

// Buffered 是 nowMs 时刻缓冲里没过期的按键数。
func (b *Buffer) Buffered(nowMs int64) int {
	b.sweepExpired(nowMs)
	return len(b.presses)
}

// Dropped 是被容量挤掉的按键数。
func (b *Buffer) Dropped() int { return b.dropped }

// Scanned 是上次匹配时看过的按键次数（规模观测）。
func (b *Buffer) Scanned() int { return b.scanned }

// Consume 用缓冲里的按键匹配一串连招，成功后把这些按键移出缓冲；
// 失败时缓冲一字不动。匹配前先按 nowMs 清掉过期按键，再沿时间顺序单遍推进，
// 每个按键最多被看一次。
func (b *Buffer) Consume(nowMs int64, chain Chain) ([]string, bool) {
	b.sweepExpired(nowMs)
	b.scanned = 0
	if len(chain.Steps) == 0 || len(b.presses) < len(chain.Steps) {
		return nil, false
	}
	stepIndex := 0
	matchedIndexes := make([]int, 0, len(chain.Steps))
	matched := make([]string, 0, len(chain.Steps))
	for index, press := range b.presses {
		b.scanned++
		if press.Action != chain.Steps[stepIndex] {
			continue
		}
		matchedIndexes = append(matchedIndexes, index)
		matched = append(matched, press.Action)
		stepIndex++
		if stepIndex == len(chain.Steps) {
			break
		}
	}
	if stepIndex < len(chain.Steps) {
		return nil, false
	}
	kept := b.presses[:0]
	matchedPos := 0
	for index, press := range b.presses {
		if matchedPos < len(matchedIndexes) && index == matchedIndexes[matchedPos] {
			matchedPos++
			continue
		}
		kept = append(kept, press)
	}
	for i := len(kept); i < len(b.presses); i++ {
		b.presses[i] = storedPress{}
	}
	b.presses = kept
	return matched, true
}

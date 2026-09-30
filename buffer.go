// Package inputbuffer 是连招输入缓冲：按时间窗口留住预输入、按时间顺序单遍匹配连招。
// 缺陷：条数上限用错小常量且不记丢弃、过期的按键不清、匹配按录入顺序、每个步骤都从头重扫缓冲。
package inputbuffer

// WindowMs 是缓冲窗口。
const WindowMs = 300

// Capacity 是缓冲允许保留的按键数上限。
const Capacity = 64

// maxPending 是缺陷里写死的错误上限。
const maxPending = 4

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
type Buffer struct {
	windowMs int64
	presses  []Press
	dropped  int
	scanned  int
}

// NewBuffer 建缓冲，windowMs 是缓冲窗口。
func NewBuffer(windowMs int64) *Buffer { return &Buffer{windowMs: windowMs} }

// Press 收一次按键。
// 缺陷：按写死的 maxPending 条裁剪（不是 Capacity），被挤掉的按键也不计入 Dropped。
func (b *Buffer) Press(action string, atMs int64) {
	b.presses = append(b.presses, Press{Action: action, AtMs: atMs})
	if len(b.presses) > maxPending {
		b.presses = b.presses[len(b.presses)-maxPending:]
	}
}

// Buffered 是 nowMs 时刻缓冲里的按键数。
// 缺陷：不看过期时间。
func (b *Buffer) Buffered(nowMs int64) int { return len(b.presses) }

// Dropped 是被容量挤掉的按键数。
// 缺陷：从来没被累加过，恒为 0。
func (b *Buffer) Dropped() int { return b.dropped }

// Scanned 是匹配时看过的按键次数（规模观测）。
func (b *Buffer) Scanned() int { return b.scanned }

// Consume 用缓冲里的按键匹配一串连招，成功后把这些按键移出缓冲。
// 缺陷：不看过期时间、不按时间戳排序，每个步骤都从缓冲头重扫一遍。
func (b *Buffer) Consume(nowMs int64, chain Chain) ([]string, bool) {
	if len(chain.Steps) == 0 || len(b.presses) < len(chain.Steps) {
		return nil, false
	}
	used := make([]bool, len(b.presses))
	matched := make([]string, 0, len(chain.Steps))
	for _, step := range chain.Steps {
		found := false
		for index := 0; index < len(b.presses); index++ {
			b.scanned++
			if used[index] || b.presses[index].Action != step {
				continue
			}
			used[index] = true
			found = true
			break
		}
		if !found {
			return nil, false
		}
		matched = append(matched, step)
	}
	rest := make([]Press, 0, len(b.presses))
	for index, press := range b.presses {
		if !used[index] {
			rest = append(rest, press)
		}
	}
	b.presses = rest
	return matched, true
}

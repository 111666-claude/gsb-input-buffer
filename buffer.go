// Package inputbuffer 是连招输入缓冲：按时间窗留住预输入，再按时间顺序消费。
// 缺陷：窗口按条数算而不是按时间，过期的按键不清，消费也不按时间顺序。
package inputbuffer

// WindowMs 是默认缓冲窗口，毫秒。
const WindowMs = 300

// MaxPending 是缺陷里写死的条数上限。
const MaxPending = 4

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
}

// NewBuffer 建缓冲，windowMs 是缓冲窗口。
func NewBuffer(windowMs int64) *Buffer {
	return &Buffer{windowMs: windowMs}
}

// Press 收一次按键。
// 缺陷：按条数裁剪，同一帧按键多了会把最早的挤出去。
func (b *Buffer) Press(action string, atMs int64) {
	b.presses = append(b.presses, Press{Action: action, AtMs: atMs})
	if len(b.presses) > MaxPending {
		b.presses = b.presses[len(b.presses)-MaxPending:]
	}
}

// Len 是缓冲里的按键数。
// 缺陷：不看过期时间，几秒前的按键也算在里面。
func (b *Buffer) Len(nowMs int64) int {
	return len(b.presses)
}

// Consume 用缓冲里的按键匹配一串连招，成功后把这些按键移出缓冲。
// 缺陷：既不看过期时间，也不按时间戳排序，直接按录入顺序匹配。
func (b *Buffer) Consume(nowMs int64, chain Chain) ([]string, bool) {
	if len(chain.Steps) == 0 || len(b.presses) < len(chain.Steps) {
		return nil, false
	}
	used := make([]bool, len(b.presses))
	cursor := 0
	matched := make([]string, 0, len(chain.Steps))
	for _, step := range chain.Steps {
		found := false
		for ; cursor < len(b.presses); cursor++ {
			if b.presses[cursor].Action == step {
				used[cursor] = true
				cursor++
				found = true
				break
			}
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

package inputbuffer

import (
	"math/rand"
	"sort"
	"testing"
)

// refEntry 是参考实现里的一条按键，seq 是全局录入序号。
type refEntry struct {
	action string
	atMs   int64
	seq    int
}

// refBuffer 是独立编写的参考实现：
// 按键按录入顺序存放；容量超出时按 (AtMs, seq) 丢最旧；
// 查询/匹配前按 now 做时间过期清理；匹配时排序后单遍推进。
type refBuffer struct {
	windowMs int64
	live     []refEntry
	dropped  int
	scanned  int
	nextSeq  int
}

func (r *refBuffer) press(action string, atMs int64) {
	r.live = append(r.live, refEntry{action: action, atMs: atMs, seq: r.nextSeq})
	r.nextSeq++
	if len(r.live) > Capacity {
		oldest := 0
		for i := 1; i < len(r.live); i++ {
			candidate, current := r.live[i], r.live[oldest]
			if candidate.atMs < current.atMs ||
				(candidate.atMs == current.atMs && candidate.seq < current.seq) {
				oldest = i
			}
		}
		r.live = append(r.live[:oldest], r.live[oldest+1:]...)
		r.dropped++
	}
}

func (r *refBuffer) sweepExpired(nowMs int64) {
	cutoff := nowMs - r.windowMs
	kept := r.live[:0]
	for _, entry := range r.live {
		if entry.atMs >= cutoff {
			kept = append(kept, entry)
		}
	}
	r.live = kept
}

func (r *refBuffer) buffered(nowMs int64) int {
	r.sweepExpired(nowMs)
	return len(r.live)
}

func (r *refBuffer) consume(nowMs int64, steps []string) ([]string, bool) {
	r.sweepExpired(nowMs)
	r.scanned = 0
	if len(steps) == 0 || len(r.live) < len(steps) {
		return nil, false
	}
	ordered := append([]refEntry(nil), r.live...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].atMs != ordered[j].atMs {
			return ordered[i].atMs < ordered[j].atMs
		}
		return ordered[i].seq < ordered[j].seq
	})
	used := make(map[int]bool, len(steps))
	matched := make([]string, 0, len(steps))
	stepIndex := 0
	for _, entry := range ordered {
		r.scanned++
		if entry.action != steps[stepIndex] {
			continue
		}
		used[entry.seq] = true
		matched = append(matched, entry.action)
		stepIndex++
		if stepIndex == len(steps) {
			break
		}
	}
	if stepIndex < len(steps) {
		return nil, false
	}
	kept := r.live[:0]
	for _, entry := range r.live {
		if !used[entry.seq] {
			kept = append(kept, entry)
		}
	}
	r.live = kept
	return matched, true
}

// diffSnapshot 是某个时刻两边缓冲剩余的可比对形态：按录入序号排序的三元组。
type diffEntry struct {
	seq    int
	atMs   int64
	action string
}

func realSnapshot(b *Buffer) []diffEntry {
	out := make([]diffEntry, 0, len(b.presses))
	for _, press := range b.presses {
		out = append(out, diffEntry{seq: press.seq, atMs: press.AtMs, action: press.Action})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].seq < out[j].seq })
	return out
}

func refSnapshot(r *refBuffer) []diffEntry {
	out := make([]diffEntry, 0, len(r.live))
	for _, entry := range r.live {
		out = append(out, diffEntry{seq: entry.seq, atMs: entry.atMs, action: entry.action})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].seq < out[j].seq })
	return out
}

// 200 组随机事件流：与独立参考实现逐操作比对消费结果、缓冲剩余与丢弃数。
func TestRandomDifferential(t *testing.T) {
	actions := []string{"light", "heavy", "a", "b"}
	for trial := 0; trial < 200; trial++ {
		rng := rand.New(rand.NewSource(int64(9000 + trial)))
		real := NewBuffer(WindowMs)
		reference := &refBuffer{windowMs: WindowMs}
		nowMs := int64(0)

		compare := func(stage string) {
			t.Helper()
			realLeft := realSnapshot(real)
			refLeft := refSnapshot(reference)
			if len(realLeft) != len(refLeft) {
				t.Fatalf("trial=%d %s 剩余条数不一致：real=%d ref=%d",
					trial, stage, len(realLeft), len(refLeft))
			}
			for i := range realLeft {
				if realLeft[i] != refLeft[i] {
					t.Fatalf("trial=%d %s 剩余按键不一致：real=%v ref=%v",
						trial, stage, realLeft, refLeft)
				}
			}
			if real.Dropped() != reference.dropped {
				t.Fatalf("trial=%d %s 丢弃数不一致：real=%d ref=%d",
					trial, stage, real.Dropped(), reference.dropped)
			}
		}

		for op := 0; op < 300; op++ {
			roll := rng.Intn(100)
			switch {
			case roll < 12: // 同刻灌入一大批，压容量
				count := 20 + rng.Intn(90)
				for i := 0; i < count; i++ {
					action := actions[rng.Intn(len(actions))]
					real.Press(action, nowMs)
					reference.press(action, nowMs)
				}
			case roll < 72:
				nowMs += int64(rng.Intn(41) - 20) // 允许乱序回退
				if nowMs < 0 {
					nowMs = 0
				}
				atMs := nowMs
				if rng.Intn(3) == 0 {
					atMs = nowMs - int64(rng.Intn(35)) // 部分按键带过去时间戳
					if atMs < 0 {
						atMs = 0
					}
				}
				action := actions[rng.Intn(len(actions))]
				real.Press(action, atMs)
				reference.press(action, atMs)
			case roll < 88:
				nowMs += int64(rng.Intn(60))
				var steps []string
				if rng.Intn(10) != 0 {
					steps = make([]string, 1+rng.Intn(4))
					for i := range steps {
						steps[i] = actions[rng.Intn(len(actions))]
					}
				}
				realMatched, realOK := real.Consume(nowMs, Chain{Steps: steps})
				refMatched, refOK := reference.consume(nowMs, steps)
				if realOK != refOK {
					t.Fatalf("trial=%d op=%d 成功与否不一致：real=%v ref=%v steps=%v",
						trial, op, realOK, refOK, steps)
				}
				if len(realMatched) != len(refMatched) {
					t.Fatalf("trial=%d op=%d 消费结果长度不一致：%v vs %v",
						trial, op, realMatched, refMatched)
				}
				for i := range realMatched {
					if realMatched[i] != refMatched[i] {
						t.Fatalf("trial=%d op=%d 消费结果不一致：%v vs %v",
							trial, op, realMatched, refMatched)
					}
				}
				if real.Scanned() > reference.bufferedSnapshotLen()+len(steps) {
					t.Fatalf("trial=%d op=%d Scanned 超出 缓冲条数+步骤数：%d",
						trial, op, real.Scanned())
				}
				if real.Scanned() != reference.scanned {
					t.Fatalf("trial=%d op=%d Scanned 不一致：real=%d ref=%d",
						trial, op, real.Scanned(), reference.scanned)
				}
				compare("consume")
			default:
				nowMs += int64(rng.Intn(60))
				if real.Buffered(nowMs) != reference.buffered(nowMs) {
					t.Fatalf("trial=%d op=%d Buffered 不一致：real=%d ref=%d",
						trial, op, real.Buffered(nowMs), len(reference.live))
				}
				compare("buffered")
			}
		}
		compare("end")
	}
}

// bufferedSnapshotLen 给出当前（不清理）缓冲条数，用于 Scanned 上限断言。
func (r *refBuffer) bufferedSnapshotLen() int { return len(r.live) }

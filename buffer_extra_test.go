package inputbuffer

import "testing"

// 过期清理：严格大于 WindowMs 才算过期，等号不算。
func TestExpiryBoundaryStrict(t *testing.T) {
	buffer := NewBuffer(WindowMs)
	buffer.Press("light", 0)
	if buffer.Buffered(WindowMs) != 1 {
		t.Fatalf("now-AtMs==WindowMs 时按键仍在窗口内：%d", buffer.Buffered(WindowMs))
	}
	if buffer.Buffered(WindowMs+1) != 0 {
		t.Fatalf("now-AtMs>WindowMs 时必须清掉：%d", buffer.Buffered(WindowMs+1))
	}
}

// 过期按键不能参与匹配；失败后缓冲仍按当前时间清空过期项。
func TestExpiredPressesDoNotMatch(t *testing.T) {
	buffer := NewBuffer(WindowMs)
	buffer.Press("light", 0)
	buffer.Press("heavy", 50)
	if matched, ok := buffer.Consume(400, Chain{Steps: []string{"light", "heavy"}}); ok {
		t.Fatalf("两个键都已过期，不该凑出连招：%v", matched)
	}
	if buffer.Buffered(400) != 0 {
		t.Fatalf("过期按键必须被丢掉：%d", buffer.Buffered(400))
	}
	if buffer.Dropped() != 0 {
		t.Fatalf("过期丢弃不计入容量丢弃数：%d", buffer.Dropped())
	}
}

// 窗口内新旧混合：过期的先清掉，没过期的照常匹配，且不影响剩余计数。
func TestConsumeSweepsOnlyExpired(t *testing.T) {
	buffer := NewBuffer(WindowMs)
	buffer.Press("light", 0)
	buffer.Press("heavy", 320)
	matched, ok := buffer.Consume(350, Chain{Steps: []string{"heavy"}})
	if !ok {
		t.Fatal("320ms 的重击在 350ms 仍在窗口内，应该能吃掉")
	}
	if len(matched) != 1 || matched[0] != "heavy" {
		t.Fatalf("吃掉的按键不对：%v", matched)
	}
	if buffer.Buffered(350) != 0 {
		t.Fatalf("过期的轻击已随清理消失，重击被吃掉，应为 0：%d", buffer.Buffered(350))
	}
}

// 容量：超出 Capacity 丢最旧一条并累加 Dropped；消费腾出空间后不再丢。
func TestCapacityDropsOldest(t *testing.T) {
	buffer := NewBuffer(WindowMs)
	for index := 0; index < Capacity; index++ {
		buffer.Press("old", int64(index))
	}
	if buffer.Dropped() != 0 || buffer.Buffered(Capacity) != Capacity {
		t.Fatalf("刚好 Capacity 条时不该丢弃：len=%d dropped=%d",
			buffer.Buffered(Capacity), buffer.Dropped())
	}
	buffer.Press("new", Capacity)
	if buffer.Dropped() != 1 {
		t.Fatalf("第 65 条应挤掉最旧一条：dropped=%d", buffer.Dropped())
	}
	if buffer.Buffered(Capacity) != Capacity {
		t.Fatalf("挤出后仍应保持 Capacity 条：%d", buffer.Buffered(Capacity))
	}
	// 最旧的 old@0 已被丢；消费一个 old（此时最旧是 old@1）成功。
	if _, ok := buffer.Consume(Capacity, Chain{Steps: []string{"old"}}); !ok {
		t.Fatal("腾出后最旧的 old@1 应仍可消费")
	}
	if buffer.Buffered(Capacity) != Capacity-1 {
		t.Fatalf("消费成功后应剩 %d 条：%d", Capacity-1, buffer.Buffered(Capacity))
	}
	buffer.Press("new2", Capacity+1)
	if buffer.Dropped() != 1 {
		t.Fatalf("有空位时不该再丢：dropped=%d", buffer.Dropped())
	}
}

// 容量按时间最旧丢：乱序到达时被挤掉的是 AtMs 最早的那条。
func TestCapacityDropsOldestByTime(t *testing.T) {
	buffer := NewBuffer(WindowMs)
	for index := 0; index < Capacity-1; index++ {
		buffer.Press("filler", int64(index+1)) // 1..63
	}
	buffer.Press("oldest", 0) // 乱序：时间最早的按键最后才录入
	buffer.Press("newest", int64(Capacity))
	if buffer.Dropped() != 1 {
		t.Fatalf("第 65 条应挤掉最旧一条：dropped=%d", buffer.Dropped())
	}
	if _, ok := buffer.Consume(200, Chain{Steps: []string{"oldest"}}); ok {
		t.Fatal("oldest@0 是时间最旧的一条，应已被容量挤掉")
	}
	if _, ok := buffer.Consume(200, Chain{Steps: []string{"newest"}}); !ok {
		t.Fatal("最新录入的按键应仍在缓冲里")
	}
	if buffer.Buffered(200) != Capacity-1 {
		t.Fatalf("消费 newest 后应剩 %d 条：%d", Capacity-1, buffer.Buffered(200))
	}
}

// 乱序到达：结果只由时间戳决定。
func TestOutOfOrderArrivalMatchesByTimestamp(t *testing.T) {
	chain := Chain{Steps: []string{"heavy", "light"}}
	first := NewBuffer(WindowMs)
	first.Press("heavy", 50)
	first.Press("light", 100)
	second := NewBuffer(WindowMs)
	second.Press("light", 100) // 事件乱序：晚发生的先到
	second.Press("heavy", 50)
	matchedFirst, okFirst := first.Consume(200, chain)
	matchedSecond, okSecond := second.Consume(200, chain)
	if okFirst != okSecond || !okFirst {
		t.Fatalf("到达顺序不应影响结果：%v/%v", okFirst, okSecond)
	}
	if len(matchedFirst) != len(matchedSecond) ||
		matchedFirst[0] != matchedSecond[0] || matchedFirst[1] != matchedSecond[1] {
		t.Fatalf("时间序相同的两串按键结果应一致：%v vs %v", matchedFirst, matchedSecond)
	}
	if first.Buffered(200) != second.Buffered(200) {
		t.Fatalf("剩余按键数应一致：%d vs %d", first.Buffered(200), second.Buffered(200))
	}
}

// 同刻按键按录入顺序：先后录入顺序相反时，靠录入序的连招结果不同。
func TestSameTimestampUsesInsertionOrder(t *testing.T) {
	chain := Chain{Steps: []string{"light", "heavy"}}
	ordered := NewBuffer(WindowMs)
	ordered.Press("light", 100)
	ordered.Press("heavy", 100)
	if _, ok := ordered.Consume(200, chain); !ok {
		t.Fatal("同刻先录 light 再录 heavy，应能按录入序凑出 light,heavy")
	}
	reversed := NewBuffer(WindowMs)
	reversed.Press("heavy", 100)
	reversed.Press("light", 100)
	if _, ok := reversed.Consume(200, chain); ok {
		t.Fatal("同刻先录 heavy 再录 light，时间序上凑不出 light,heavy")
	}
	if reversed.Buffered(200) != 2 {
		t.Fatalf("匹配失败时缓冲必须一字不动：%d", reversed.Buffered(200))
	}
}

// 匹配失败时缓冲一字不动，包括没有消费任何按键与丢弃数不变。
func TestFailedConsumeLeavesBufferUntouched(t *testing.T) {
	buffer := NewBuffer(WindowMs)
	buffer.Press("light", 10)
	buffer.Press("light", 20)
	buffer.Press("heavy", 30)
	snapshot := append([]storedPress(nil), buffer.presses...)
	if _, ok := buffer.Consume(100, Chain{Steps: []string{"heavy", "light"}}); ok {
		t.Fatal("时间序上 heavy 在最后，凑不出 heavy,light")
	}
	if len(buffer.presses) != len(snapshot) {
		t.Fatalf("失败后缓冲条数变了：%d vs %d", len(buffer.presses), len(snapshot))
	}
	for index := range snapshot {
		if buffer.presses[index] != snapshot[index] {
			t.Fatalf("失败后缓冲内容变了：%v vs %v", buffer.presses, snapshot)
		}
	}
	if buffer.Scanned() > len(snapshot) {
		t.Fatalf("单遍扫描每个按键最多看一次：scanned=%d len=%d",
			buffer.Scanned(), len(snapshot))
	}
}

// Scanned 不超过 缓冲条数 + 步骤数，且不超过缓冲条数（单遍）。
func TestScannedBound(t *testing.T) {
	buffer := NewBuffer(WindowMs)
	for index := 0; index < 20; index++ {
		buffer.Press("light", int64(index))
	}
	buffer.Consume(20, Chain{Steps: []string{"light", "heavy", "light"}})
	if buffer.Scanned() > 20+3 {
		t.Fatalf("Scanned 超出 缓冲条数+步骤数：%d", buffer.Scanned())
	}
	if buffer.Scanned() > 20 {
		t.Fatalf("游标单遍推进，每个按键最多看一次：%d", buffer.Scanned())
	}
}

// 成功消费后 Buffered 正好少掉吃掉的步骤数。
func TestConsumeRemovesExactlyMatched(t *testing.T) {
	buffer := NewBuffer(WindowMs)
	for index := 0; index < 10; index++ {
		buffer.Press("light", int64(index))
	}
	matched, ok := buffer.Consume(10, Chain{Steps: []string{"light", "light", "light"}})
	if !ok || len(matched) != 3 {
		t.Fatalf("应吃掉三个 light：%v %v", matched, ok)
	}
	if buffer.Buffered(10) != 7 {
		t.Fatalf("成功后应正好少 3 条：%d", buffer.Buffered(10))
	}
	// 重复回放结果相同。
	again, okAgain := buffer.Consume(10, Chain{Steps: []string{"light", "light", "light"}})
	if !okAgain || len(again) != 3 || buffer.Buffered(10) != 4 {
		t.Fatalf("重复回放应继续吃掉 3 条：%v %v len=%d", again, okAgain, buffer.Buffered(10))
	}
}

// 空连招与缓冲不足时直接失败，Scanned 为 0。
func TestEmptyAndShortChain(t *testing.T) {
	buffer := NewBuffer(WindowMs)
	buffer.Press("light", 0)
	if _, ok := buffer.Consume(10, Chain{}); ok {
		t.Fatal("空连招不该成功")
	}
	if buffer.Scanned() != 0 {
		t.Fatalf("空连招不该扫描任何按键：%d", buffer.Scanned())
	}
	if _, ok := buffer.Consume(10, Chain{Steps: []string{"light", "heavy"}}); ok {
		t.Fatal("只有一个键时凑不出两步连招")
	}
	if buffer.Buffered(10) != 1 {
		t.Fatalf("失败缓冲不动：%d", buffer.Buffered(10))
	}
}

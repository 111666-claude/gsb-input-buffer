package inputbuffer

import "testing"

func TestConsumeInOrder(t *testing.T) {
	buffer := NewBuffer(WindowMs)
	buffer.Press("light", 0)
	buffer.Press("light", 100)
	buffer.Press("heavy", 150)
	matched, ok := buffer.Consume(200, Chain{Steps: []string{"light", "heavy"}})
	if !ok {
		t.Fatal("按顺序按下的连招应该能吃掉")
	}
	if len(matched) != 2 || matched[0] != "light" || matched[1] != "heavy" {
		t.Fatalf("吃掉的步骤不对：%v", matched)
	}
	if buffer.Len(200) != 1 {
		t.Fatalf("吃掉两个键之后应该剩一个：%d", buffer.Len(200))
	}
}

func TestConsumeFailsWithoutSteps(t *testing.T) {
	buffer := NewBuffer(WindowMs)
	buffer.Press("light", 0)
	if _, ok := buffer.Consume(50, Chain{Steps: []string{"heavy"}}); ok {
		t.Fatal("没有按过的键不该被吃掉")
	}
}

func TestLenCountsFreshPresses(t *testing.T) {
	buffer := NewBuffer(WindowMs)
	buffer.Press("light", 1000)
	buffer.Press("heavy", 1050)
	if buffer.Len(1100) != 2 {
		t.Fatalf("窗口内的按键应该都还在：%d", buffer.Len(1100))
	}
}

func TestConstants(t *testing.T) {
	if WindowMs != 300 || MaxPending != 4 {
		t.Fatal("常量被改了")
	}
}

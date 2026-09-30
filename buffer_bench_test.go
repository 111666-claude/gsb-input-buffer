package inputbuffer

import "testing"

// BenchmarkPressAtCapacity：缓冲已满时持续按键，观测按键吞吐（目标每秒 10 万次以上）。
func BenchmarkPressAtCapacity(b *testing.B) {
	buffer := NewBuffer(WindowMs)
	for index := 0; index < Capacity; index++ {
		buffer.Press("warm", int64(index))
	}
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		buffer.Press("x", int64(index))
	}
}

// BenchmarkConsumeFullBuffer：满缓冲下高频匹配（多数失败）。
func BenchmarkConsumeFullBuffer(b *testing.B) {
	chain := Chain{Steps: []string{"heavy", "light", "heavy", "light"}}
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		buffer := NewBuffer(WindowMs)
		for j := 0; j < Capacity; j++ {
			buffer.Press("light", int64(j))
		}
		buffer.Consume(100, chain)
	}
}

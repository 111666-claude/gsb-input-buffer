// Command input-buffer 跑连招输入缓冲样例。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"example.com/inputbuffer"
)

func report(stdout io.Writer, buffer *inputbuffer.Buffer, nowMs int64, matched []string, ok bool) {
	result := "none"
	if ok {
		result = strings.Join(matched, ",")
	}
	fmt.Fprintf(stdout, "result=%s len=%d dropped=%d\n", result, buffer.Buffered(nowMs), buffer.Dropped())
}

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("input-buffer", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "expired", "expired / crowded / order / overflow")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	buffer := inputbuffer.NewBuffer(inputbuffer.WindowMs)
	chain := inputbuffer.Chain{Steps: []string{"light", "heavy"}}
	switch *sample {
	case "expired":
		// 玩家五秒前按了两下，现在才轮到输出判定。
		buffer.Press("light", 0)
		buffer.Press("heavy", 0)
		matched, ok := buffer.Consume(5000, chain)
		report(stdout, buffer, 5000, matched, ok)
	case "crowded":
		// 同一帧里按下五个键，第一个是起手的重击。
		chain = inputbuffer.Chain{Steps: []string{"heavy", "light"}}
		buffer.Press("heavy", 0)
		for index := 0; index < 4; index++ {
			buffer.Press("light", 0)
		}
		matched, ok := buffer.Consume(100, chain)
		report(stdout, buffer, 100, matched, ok)
	case "order":
		// 事件乱序到达：先录到 100ms 的轻击，再录到 50ms 的重击。
		buffer.Press("light", 100)
		buffer.Press("heavy", 50)
		matched, ok := buffer.Consume(200, chain)
		report(stdout, buffer, 200, matched, ok)
	case "overflow":
		// 同一帧里灌进一百个按键。
		for index := 0; index < 100; index++ {
			buffer.Press(fmt.Sprintf("key-%d", index), 0)
		}
		fmt.Fprintf(stdout, "len=%d dropped=%d scanned=%d\n",
			buffer.Buffered(0), buffer.Dropped(), buffer.Scanned())
	default:
		fmt.Fprintln(stderr, "需要 --sample expired|crowded|order|overflow")
		return 2
	}
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}

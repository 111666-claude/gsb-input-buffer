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

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("input-buffer", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "expired", "expired / crowded / order")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	buffer := inputbuffer.NewBuffer(inputbuffer.WindowMs)
	chain := inputbuffer.Chain{Steps: []string{"light", "heavy"}}
	nowMs := int64(0)
	switch *sample {
	case "expired":
		// 玩家五秒前按了两下，现在才轮到输出判定。
		buffer.Press("light", 0)
		buffer.Press("heavy", 0)
		nowMs = 5000
	case "crowded":
		// 同一帧里按下五个键，第一个是起手的重击，连招是重击接轻击。
		chain = inputbuffer.Chain{Steps: []string{"heavy", "light"}}
		buffer.Press("heavy", 0)
		for index := 0; index < 4; index++ {
			buffer.Press("light", 0)
		}
		nowMs = 100
	case "order":
		// 事件乱序到达：重击的时间戳比重击之后的轻击还早。
		buffer.Press("light", 100)
		buffer.Press("heavy", 50)
		nowMs = 200
	default:
		fmt.Fprintln(stderr, "需要 --sample expired|crowded|order")
		return 2
	}
	matched, ok := buffer.Consume(nowMs, chain)
	result := "none"
	if ok {
		result = strings.Join(matched, ",")
	}
	fmt.Fprintf(stdout, "result=%s len=%d\n", result, buffer.Len(nowMs))
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}

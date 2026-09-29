# input-buffer

连招输入缓冲：按时间窗留住预输入，再按时间顺序消费，只用 Go 标准库。

```
go test ./...
go run ./cmd/input-buffer --sample expired
go run ./cmd/input-buffer --sample crowded
go run ./cmd/input-buffer --sample order
```

## 口径（README 为准）

- **窗口按时间算**：`nowMs - AtMs > WindowMs`（严格大于）的按键必须被丢掉，
  过期的按键不能参与匹配。`--sample expired`（0ms 按下的两个键、5000ms 才判定）
  按口径是 `result=none len=0`，现在是 `result=light,heavy len=0`。
- **条数不该有硬上限**：同一帧里按下的按键只要没过期就都算数。
  `--sample crowded`（同一帧先按重击再按四个轻击，连招是重击接轻击）按口径是
  `result=heavy,light len=3`（吃掉两个，还剩三个轻击），现在因为只留 4 条把最早的重击挤掉了，得到
  `result=none len=4`。
- **消费按时间顺序**：匹配前先按 `AtMs` 升序排序，`AtMs` 相同按录入顺序；
  匹配成功才把这些按键移出缓冲，匹配失败不动缓冲。
  `--sample order`（先录到 100ms 的轻击、再录到 50ms 的重击）按口径是
  `result=none len=2`（时间顺序里重击在轻击之前，连招凑不出来），现在是
  `result=light,heavy len=0`。
- 不变量：`Consume` 成功后 `Len` 正好少掉吃掉的步骤数，失败时缓冲一字不动；
  同一串按键事件无论到达顺序如何，结果只由时间戳决定；重复回放结果相同。
- 规模：单次按键摊还 O(1)，单次 `Consume` O(缓冲条数 + 步骤数)，内存只跟窗口内的按键数
  有关；不许把整局按键都留着。

## 现在的行为

- `Press` 超过 `MaxPending = 4` 条就丢最早的，帧率一变同一个输入会被吞。
- `Len` 与 `Consume` 都不看过期时间，几秒前的按键还能凑出连招。
- `Consume` 按录入顺序匹配，事件乱序到达就会认错。

## 输出

```
result=none len=0
result=light,heavy len=3
result=none len=2
```

## 目录

```
buffer.go               输入缓冲
cmd/input-buffer        命令行入口
buffer_test.go          go test 用例
```

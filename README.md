# input-buffer

连招输入缓冲：按时间窗口留住预输入、按时间顺序单遍匹配连招，只用 Go 标准库。

```
go test ./...
go run ./cmd/input-buffer --sample expired
go run ./cmd/input-buffer --sample crowded
go run ./cmd/input-buffer --sample order
go run ./cmd/input-buffer --sample overflow
```

## 口径（README 为准）

- **窗口按时间算**：`now - AtMs > WindowMs`（严格大于）的按键必须丢掉，
  过期按键不能参与匹配；`Buffered(now)` 只数没过期的。
- **容量**：缓冲最多保留 `Capacity` 条按键，超出的丢最旧的那条，
  并把丢弃次数累加到 `Dropped()`。
- **匹配按时间顺序**：匹配前按 `AtMs` 升序排序（同刻按录入顺序），
  游标单遍推进：每个按键最多被看一次，`Scanned()` 不超过 `缓冲条数 + 步骤数`。
- **消费语义**：匹配成功才把这些按键移出缓冲，失败时缓冲一字不动。
- **不变量**：`Consume` 成功后 `Buffered` 正好少掉吃掉的步骤数；同一串按键无论到达顺序如何，
  结果只由时间戳决定；重复回放结果相同。
- **规模**：每秒 10 万次按键、每秒 1 万次匹配；内存只跟窗口内的按键数有关，
  不许把整局按键都留着。

## 输出契约（不改格式）

```
result=none len=0 dropped=0
result=heavy,light len=3 dropped=0
result=none len=2 dropped=0
len=64 dropped=36 scanned=0
```

## 常量

```
WindowMs = 300
Capacity = 64
```

## 目录

```
buffer.go               输入缓冲
cmd/input-buffer        命令行入口
buffer_test.go          go test 用例
```

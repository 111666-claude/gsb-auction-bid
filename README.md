# auction-bid

拍卖场：出价、最小加价、最后 10 秒延时与押金，只用 Go 标准库。

```
go test ./...
go run ./cmd/auction-bid --sample minstep
go run ./cmd/auction-bid --sample deposits
```

## 口径（README 为准）

- **最小加价要严格大于**：新出价必须 `> 当前价 + 最小加价`。
  `--sample minstep`（当前价 100、最小加价 10，出价 110）按口径是 `accepted=false`，现在给 `accepted=true`。
- **落槌要退押金**：除赢家外的所有押金要退回。`--sample deposits`（3 人各押 50，b 以 300 成交）
  按口径是 `refunded=2`，现在是 0。
- **最后 10 秒延时**：若出价发生在 `EndsAt-10` 秒之后，结束时间顺延 10 秒（现在完全没实现）。
- 不变量：成交价单调递增；成交者必须有押金且是最后一次有效出价的人；超时后的出价一律不接受；
  同一批出价重复执行结果相同。
- 规模：10 万个拍品、每个 100 次出价 1 秒内处理完，内存 O(拍品 + 押金人数)；不许每次出价都全表扫描。

## 现在的行为

- `Bid` 的条件是 `amount >= l.Price+l.MinStep`：等于最低加价也被接受。
- `Bid` 完全不看 `EndsAt-10`，没有延时逻辑。
- `Settle` 把 `refunded` 写死成 0。

## 输出

```
accepted=false price=100
winner=b price=300 refunded=2
```

## 目录

```
lot.go              出价、延时与押金
cmd/auction-bid     命令行入口
lot_test.go         go test 用例
```

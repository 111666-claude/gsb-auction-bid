// Package auction 是拍卖场：出价、最小加价、延时与押金。
package auction

import "errors"

// ErrInput 表示输入不合法。
var ErrInput = errors.New("auction: 输入不合法")

// Lot 是一件拍品。
type Lot struct {
	Price    float64
	MinStep  float64
	Bidder   string
	EndsAt   int64
	Deposits map[string]float64
}

// New 建拍品。
func New(price, step float64, endsAt int64) *Lot {
	if step <= 0 {
		step = 1
	}
	return &Lot{Price: price, MinStep: step, EndsAt: endsAt, Deposits: map[string]float64{}}
}

// Deposit 缴纳押金。
func (l *Lot) Deposit(bidder string, amount float64) error {
	if amount <= 0 {
		return ErrInput
	}
	l.Deposits[bidder] = amount
	return nil
}

// Bid 出价。新出价必须严格大于「当前价 + 最小加价」；
// 若有效出价发生在 EndsAt-10 秒之后，结束时间顺延 10 秒。
func (l *Lot) Bid(bidder string, amount float64, now int64) (bool, error) {
	if amount <= 0 || bidder == "" {
		return false, ErrInput
	}
	if now > l.EndsAt {
		return false, nil
	}
	if amount > l.Price+l.MinStep {
		l.Price = amount
		l.Bidder = bidder
		if now > l.EndsAt-10 {
			l.EndsAt += 10
		}
		return true, nil
	}
	return false, nil
}

// Settle 落槌。除赢家外的押金全部退回，返回赢家、成交价与退回人数。
// 重复调用幂等：已退回的押金不会重复统计。
func (l *Lot) Settle(now int64) (string, float64, int) {
	refunded := 0
	for bidder := range l.Deposits {
		if bidder == l.Bidder {
			continue
		}
		delete(l.Deposits, bidder)
		refunded++
	}
	return l.Bidder, l.Price, refunded
}

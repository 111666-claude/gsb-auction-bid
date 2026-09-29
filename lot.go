// Package auction 是拍卖场：出价、最小加价、延时与押金。
// 缺陷：最小加价用 >=（等于最低加价也接受）、最后 10 秒出价不延时、落槌后退押金没实现。
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

// Bid 出价。缺陷：刚好等于「当前价 + 最小加价」也接受，也不做最后 10 秒延时。
func (l *Lot) Bid(bidder string, amount float64, now int64) (bool, error) {
	if amount <= 0 || bidder == "" {
		return false, ErrInput
	}
	if now > l.EndsAt {
		return false, nil
	}
	if amount >= l.Price+l.MinStep {
		l.Price = amount
		l.Bidder = bidder
		return true, nil
	}
	return false, nil
}

// Settle 落槌。缺陷：直接返回赢家与成交价，输家的押金不退。
func (l *Lot) Settle(now int64) (string, float64, int) {
	refunded := 0
	return l.Bidder, l.Price, refunded
}

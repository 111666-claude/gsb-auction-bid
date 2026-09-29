package auction

import (
	"math/rand"
	"testing"
	"time"
)

func TestMinStepMustBeStrictlyGreater(t *testing.T) {
	lot := New(100, 10, 1000)
	if ok, _ := lot.Bid("a", 110, 10); ok {
		t.Fatal("刚好等于「当前价 + 最小加价」的出价必须拒绝")
	}
	if lot.Price != 100 || lot.Bidder != "" {
		t.Fatalf("被拒绝的出价不该改状态：price=%v bidder=%q", lot.Price, lot.Bidder)
	}
	if ok, _ := lot.Bid("a", 110.01, 11); !ok {
		t.Fatal("严格大于最低加价的出价应该接受")
	}
	if lot.Price != 110.01 || lot.Bidder != "a" {
		t.Fatalf("有效出价没生效：price=%v bidder=%q", lot.Price, lot.Bidder)
	}
}

func TestLastTenSecondsExtend(t *testing.T) {
	lot := New(100, 10, 100)
	// 恰好在 EndsAt-10 出价：不触发延时。
	if ok, _ := lot.Bid("a", 200, 90); !ok {
		t.Fatal("结束前的有效出价应该接受")
	}
	if lot.EndsAt != 100 {
		t.Fatalf("EndsAt-10 时刻出价不该延时：EndsAt=%v", lot.EndsAt)
	}
	// EndsAt-10 之后出价：顺延 10 秒。
	if ok, _ := lot.Bid("b", 300, 95); !ok {
		t.Fatal("压哨出价应该接受")
	}
	if lot.EndsAt != 110 {
		t.Fatalf("最后 10 秒出价应顺延 10 秒：EndsAt=%v", lot.EndsAt)
	}
	// 原结束时间之后、新结束时间之前的出价现在可以接受。
	if ok, _ := lot.Bid("a", 400, 105); !ok {
		t.Fatal("延时后窗口内的出价应该接受")
	}
	// 连续压哨会再次顺延。
	if lot.EndsAt != 120 {
		t.Fatalf("105 秒出价应再次顺延：EndsAt=%v", lot.EndsAt)
	}
	if ok, _ := lot.Bid("b", 500, 121); ok {
		t.Fatal("超过顺延后的结束时间不该接受")
	}
}

func TestSettleRefundsNonWinners(t *testing.T) {
	lot := New(100, 10, 1000)
	for _, bidder := range []string{"a", "b", "c"} {
		if err := lot.Deposit(bidder, 50); err != nil {
			t.Fatalf("缴押金失败：%v", err)
		}
	}
	lot.Bid("a", 150, 10)
	lot.Bid("b", 300, 20)
	winner, price, refunded := lot.Settle(1000)
	if winner != "b" || price != 300 || refunded != 2 {
		t.Fatalf("落槌结果不对：winner=%q price=%v refunded=%v", winner, price, refunded)
	}
	if len(lot.Deposits) != 1 || lot.Deposits["b"] != 50 {
		t.Fatalf("落槌后应只剩赢家的押金：%v", lot.Deposits)
	}
	// 重复落槌幂等：不会重复退押金。
	if _, _, again := lot.Settle(1000); again != 0 {
		t.Fatalf("重复落槌不该再退押金：%v", again)
	}
}

func TestReplayDeterministic(t *testing.T) {
	run := func() (string, float64, int) {
		lot := New(100, 10, 100)
		lot.Deposit("a", 50)
		lot.Deposit("b", 50)
		lot.Bid("a", 120, 10)
		lot.Bid("b", 130, 95) // 压哨，顺延到 110
		lot.Bid("a", 140, 105)
		lot.Bid("b", 150, 200) // 超时，拒绝
		return lot.Settle(110)
	}
	w1, p1, r1 := run()
	w2, p2, r2 := run()
	if w1 != w2 || p1 != p2 || r1 != r2 {
		t.Fatalf("同一批出价重复执行结果不同：(%q,%v,%v) vs (%q,%v,%v)", w1, p1, r1, w2, p2, r2)
	}
}

func TestScaleManyLots(t *testing.T) {
	const lots, bidsPerLot = 100000, 100
	start := time.Now()
	for i := 0; i < lots; i++ {
		lot := New(100, 1, 1000)
		lot.Deposit("a", 50)
		lot.Deposit("b", 50)
		price := 100.0
		for j := 0; j < bidsPerLot; j++ {
			price += 2
			lot.Bid("ab"[j%2:j%2+1], price, int64(j))
		}
		lot.Settle(1000)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("10 万拍品 x 100 次出价应在 1 秒内完成，实际 %v", elapsed)
	} else {
		t.Logf("10 万拍品 x 100 次出价耗时 %v", elapsed)
	}
}

// refLot 是参照实现：严格加价 + 最后 10 秒延时 + 落槌退押金，刻意独立于 Lot 编写。
type refLot struct {
	price    float64
	step     float64
	bidder   string
	endsAt   int64
	deposits map[string]float64
}

func (r *refLot) bid(bidder string, amount float64, now int64) bool {
	if amount <= 0 || bidder == "" || now > r.endsAt || amount <= r.price+r.step {
		return false
	}
	r.price = amount
	r.bidder = bidder
	if now > r.endsAt-10 {
		r.endsAt += 10
	}
	return true
}

func (r *refLot) settle() (string, float64, int) {
	refunded := 0
	for bidder := range r.deposits {
		if bidder != r.bidder {
			refunded++
		}
	}
	return r.bidder, r.price, refunded
}

func TestAgainstReference(t *testing.T) {
	rng := rand.New(rand.NewSource(20260930))
	for trial := 0; trial < 200; trial++ {
		base := float64(rng.Intn(900) + 100)
		step := float64(rng.Intn(20) + 1)
		endsAt := int64(rng.Intn(900) + 100)

		lot := New(base, step, endsAt)
		ref := &refLot{price: base, step: step, endsAt: endsAt, deposits: map[string]float64{}}

		bidders := []string{"a", "b", "c", "d", "e"}[:rng.Intn(4)+2]
		for _, bidder := range bidders {
			amount := float64(rng.Intn(90) + 10)
			if err := lot.Deposit(bidder, amount); err != nil {
				t.Fatalf("trial %d：缴押金失败：%v", trial, err)
			}
			ref.deposits[bidder] = amount
		}

		for i := 0; i < 60; i++ {
			bidder := bidders[rng.Intn(len(bidders))]
			amount := base + float64(rng.Intn(2000))
			var now int64
			switch rng.Intn(3) {
			case 0: // 压哨出价：落在 EndsAt-10 附近
				now = ref.endsAt - int64(rng.Intn(25))
			default:
				now = int64(rng.Intn(int(endsAt) + 100))
			}
			if now < 0 {
				now = 0
			}

			gotOK, gotErr := lot.Bid(bidder, amount, now)
			wantOK := ref.bid(bidder, amount, now)
			if gotErr != nil || gotOK != wantOK {
				t.Fatalf("trial %d 第 %d 笔：Bid(%q,%v,%v) 得到 (%v,%v)，参考为 %v",
					trial, i, bidder, amount, now, gotOK, gotErr, wantOK)
			}
			if lot.Price != ref.price || lot.Bidder != ref.bidder || lot.EndsAt != ref.endsAt {
				t.Fatalf("trial %d 第 %d 笔后状态不一致：got(price=%v bidder=%q endsAt=%v) want(price=%v bidder=%q endsAt=%v)",
					trial, i, lot.Price, lot.Bidder, lot.EndsAt, ref.price, ref.bidder, ref.endsAt)
			}
		}

		winner, price, refunded := lot.Settle(ref.endsAt)
		refWinner, refPrice, refRefunded := ref.settle()
		if winner != refWinner || price != refPrice || refunded != refRefunded {
			t.Fatalf("trial %d 落槌不一致：got(%q,%v,%v) want(%q,%v,%v)",
				trial, winner, price, refunded, refWinner, refPrice, refRefunded)
		}
		if winner != "" {
			if _, ok := ref.deposits[winner]; !ok {
				t.Fatalf("trial %d：成交者 %q 必须有押金", trial, winner)
			}
		}
	}
}

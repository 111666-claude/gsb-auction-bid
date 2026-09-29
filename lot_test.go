package auction

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestFirstBidWins(t *testing.T) {
	lot := New(100, 10, 1000)
	ok, err := lot.Bid("a", 200, 10)
	if err != nil || !ok {
		t.Fatalf("第一次出价应该成功：%v %v", ok, err)
	}
	if lot.Bidder != "a" || lot.Price != 200 {
		t.Fatalf("赢家与价格不对：%v %v", lot.Bidder, lot.Price)
	}
}

func TestBidAfterEndFails(t *testing.T) {
	lot := New(100, 10, 100)
	if ok, _ := lot.Bid("a", 500, 200); ok {
		t.Fatal("结束之后不该接受出价")
	}
}

func TestDepositRecords(t *testing.T) {
	lot := New(100, 10, 1000)
	lot.Deposit("a", 50)
	if lot.Deposits["a"] != 50 {
		t.Fatalf("押金没记上：%v", lot.Deposits)
	}
	if err := lot.Deposit("b", 0); err != ErrInput {
		t.Fatalf("非法押金应该报错：%v", err)
	}
}

func TestBadBidInput(t *testing.T) {
	if _, err := New(100, 10, 1000).Bid("", 200, 10); err != ErrInput {
		t.Fatalf("空出价人应该报错：%v", err)
	}
}

func TestMinStepStrictBoundary(t *testing.T) {
	lot := New(100, 10, 1000)
	if ok, _ := lot.Bid("a", 110, 10); ok {
		t.Fatal("刚好等于最低加价不该被接受")
	}
	if lot.Price != 100 || lot.Bidder != "" {
		t.Fatalf("被拒绝的出价不该改状态：%v %q", lot.Price, lot.Bidder)
	}
	if ok, _ := lot.Bid("a", 110.01, 10); !ok {
		t.Fatal("严格大于最低加价应该被接受")
	}
	if lot.Price != 110.01 || lot.Bidder != "a" {
		t.Fatalf("有效出价没生效：%v %q", lot.Price, lot.Bidder)
	}
}

func TestLastTenSecondsExtension(t *testing.T) {
	lot := New(100, 10, 1000)
	if ok, _ := lot.Bid("a", 200, 995); !ok {
		t.Fatal("压哨出价应该被接受")
	}
	if lot.EndsAt != 1010 {
		t.Fatalf("结束时间应该顺延 10 秒：%v", lot.EndsAt)
	}
	if ok, _ := lot.Bid("b", 300, 1005); !ok {
		t.Fatal("顺延窗口内应该还能出价")
	}
	if lot.EndsAt != 1020 {
		t.Fatalf("顺延窗口内出价应该再次顺延：%v", lot.EndsAt)
	}

	early := New(100, 10, 1000)
	early.Bid("a", 200, 100)
	if early.EndsAt != 1000 {
		t.Fatalf("非压哨出价不该延时：%v", early.EndsAt)
	}

	edge := New(100, 10, 1000)
	edge.Bid("a", 200, 990)
	if edge.EndsAt != 1000 {
		t.Fatalf("恰好 EndsAt-10 不算「之后」，不该延时：%v", edge.EndsAt)
	}
}

func TestSettleRefundsDeposits(t *testing.T) {
	lot := New(100, 10, 1000)
	lot.Deposit("a", 50)
	lot.Deposit("b", 50)
	lot.Deposit("c", 50)
	lot.Bid("a", 150, 10)
	lot.Bid("b", 300, 20)
	winner, price, refunded := lot.Settle(30)
	if winner != "b" || price != 300 || refunded != 2 {
		t.Fatalf("落槌结果不对：%v %v %v", winner, price, refunded)
	}
	if _, ok := lot.Deposits["a"]; ok {
		t.Fatal("输家 a 的押金应该已退回")
	}
	if _, ok := lot.Deposits["c"]; ok {
		t.Fatal("输家 c 的押金应该已退回")
	}
	if _, ok := lot.Deposits["b"]; !ok {
		t.Fatal("赢家的押金不该退")
	}
}

// reference 是「严格加价 + 延时 + 退押金」的独立参考实现，用于逐步对照。
type reference struct {
	price    float64
	step     float64
	bidder   string
	endsAt   int64
	deposits map[string]float64
}

func newReference(price, step float64, endsAt int64) *reference {
	return &reference{price: price, step: step, endsAt: endsAt, deposits: map[string]float64{}}
}

func (r *reference) deposit(bidder string, amount float64) {
	r.deposits[bidder] = amount
}

func (r *reference) bid(bidder string, amount float64, now int64) bool {
	if now > r.endsAt || amount <= r.price+r.step {
		return false
	}
	r.price = amount
	r.bidder = bidder
	if now > r.endsAt-10 {
		r.endsAt += 10
	}
	return true
}

func (r *reference) settle() (string, float64, int) {
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
	bidders := []string{"a", "b", "c", "d", "e"}
	for trial := 0; trial < 200; trial++ {
		price := float64(rng.Intn(900) + 100)
		step := float64(rng.Intn(20) + 1)
		endsAt := int64(rng.Intn(500) + 100)
		lot := New(price, step, endsAt)
		ref := newReference(price, step, endsAt)
		replay := New(price, step, endsAt)

		for _, bidder := range bidders {
			if rng.Intn(2) == 0 {
				amount := float64(rng.Intn(90) + 10)
				lot.Deposit(bidder, amount)
				ref.deposit(bidder, amount)
				replay.Deposit(bidder, amount)
			}
		}

		type event struct {
			bidder string
			amount float64
			now    int64
		}
		var events []event
		for i := 0; i < 40; i++ {
			ev := event{
				bidder: bidders[rng.Intn(len(bidders))],
				amount: price + float64(rng.Intn(400)),
				now:    int64(rng.Intn(int(endsAt) + 40)),
			}
			if i%5 == 4 { // 压哨出价
				ev.now = endsAt - int64(rng.Intn(12))
			}
			events = append(events, ev)
		}

		for i, ev := range events {
			ok, err := lot.Bid(ev.bidder, ev.amount, ev.now)
			if err != nil {
				t.Fatalf("trial %d bid %d: 意外报错：%v", trial, i, err)
			}
			want := ref.bid(ev.bidder, ev.amount, ev.now)
			if ok != want || lot.Price != ref.price || lot.Bidder != ref.bidder || lot.EndsAt != ref.endsAt {
				t.Fatalf("trial %d bid %d 分歧：got (%v %v %q %v) want (%v %v %q %v)",
					trial, i, ok, lot.Price, lot.Bidder, lot.EndsAt, want, ref.price, ref.bidder, ref.endsAt)
			}
			replay.Bid(ev.bidder, ev.amount, ev.now)
		}

		winner, finalPrice, refunded := lot.Settle(endsAt + 100)
		wantWinner, wantPrice, wantRefunded := ref.settle()
		if winner != wantWinner || finalPrice != wantPrice || refunded != wantRefunded {
			t.Fatalf("trial %d 落槌分歧：got (%q %v %d) want (%q %v %d)",
				trial, winner, finalPrice, refunded, wantWinner, wantPrice, wantRefunded)
		}

		// 同一批出价重复执行结果相同。
		replayWinner, replayPrice, replayRefunded := replay.Settle(endsAt + 100)
		if replayWinner != winner || replayPrice != finalPrice || replayRefunded != refunded {
			t.Fatalf("trial %d 重复执行结果不同：(%q %v %d) vs (%q %v %d)",
				trial, replayWinner, replayPrice, replayRefunded, winner, finalPrice, refunded)
		}
	}
}

// 规模：10 万个拍品、每个 100 次出价。
func BenchmarkScale(b *testing.B) {
	const lots, bidsPerLot = 100000, 100
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		items := make([]*Lot, lots)
		for i := range items {
			items[i] = New(100, 10, 1<<60)
			items[i].Deposit(fmt.Sprintf("bidder-%d", i), 50)
		}
		b.StartTimer()
		for _, lot := range items {
			for j := 0; j < bidsPerLot; j++ {
				lot.Bid("bidder", 100+float64(j+1)*11, int64(j))
			}
			lot.Settle(bidsPerLot)
		}
	}
}

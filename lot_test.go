package auction

import "testing"

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

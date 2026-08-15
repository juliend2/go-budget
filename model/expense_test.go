package model_test

import (
	"testing"
	"time"

	"desrosiers.org/budget/model"
	"github.com/dromara/carbon/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestIsDue(t *testing.T) {
	exp := model.NewExpense(
		50,
		time.Now().Add(time.Hour*24*10),
		model.WithNow(time.Now().Add(time.Hour*24*1)),
	)
	got := exp.IsDue()
	if got == true {
		t.Errorf("IsDue() = %v; want false", got)
	}
}

func TestIsPaid(t *testing.T) {
	// Unpaid
	exp := model.NewExpense(100, time.Now())
	if exp.IsPaid() {
		t.Errorf("Expected new expense to be unpaid")
	}

	// Partially paid
	exp.Payments = []model.Payment{
		{Amount: 40},
		{Amount: 30},
	}
	if exp.IsPaid() {
		t.Errorf("Expected expense with total payments 70 < 100 to be unpaid")
	}

	// Fully paid
	exp.Payments = append(exp.Payments, model.Payment{Amount: 30})
	if !exp.IsPaid() {
		t.Errorf("Expected expense with total payments 100 >= 100 to be paid")
	}
}

func TestGetPayDays(t *testing.T) {
	// Arrange
	dateRange := model.DateRange{
		From: model.Date(2025, time.December, 1),
		To:   model.Date(2026, time.January, 15),
	}

	// Act
	payDays := model.GetPayDays(dateRange)

	// Assert
	if len(payDays) != 3 {
		t.Errorf("len(GetPayDays()) = %d; want 3", len(payDays))
	}

	got := carbon.NewCarbon(payDays[0]).ToDateString()
	if got != "2025-12-15" {
		t.Errorf("payDays[0] = %s; want 2025-12-15", got)
	}

	got = carbon.NewCarbon(payDays[1]).ToDateString()
	if got != "2025-12-31" {
		t.Errorf("payDays[1] = %s; want 2025-12-31", got)
	}

	got = carbon.NewCarbon(payDays[2]).ToDateString()
	if got != "2026-01-15" {
		t.Errorf("payDays[2] = %s; want 2026-01-15", got)
	}
}

func TestPutExpensesInTheirPayPeriods(t *testing.T) {
	// Arrange
	payDays := []time.Time{
		model.Date(2026, time.June, 30),
		model.Date(2026, time.July, 15),
		model.Date(2026, time.July, 31),
	}

	expenses := []*model.Expense{
		// first pay period: [2026-06-30, 2026-07-15)
		model.NewExpense(5, model.Date(2026, time.June, 30)),  // due on the pay day -> belongs here
		model.NewExpense(10, model.Date(2026, time.July, 1)),
		model.NewExpense(15, model.Date(2026, time.July, 14)),
		// second pay period: [2026-07-15, 2026-07-31)
		model.NewExpense(20, model.Date(2026, time.July, 15)), // due on the pay day -> belongs here
		model.NewExpense(30, model.Date(2026, time.July, 16)),
		model.NewExpense(40, model.Date(2026, time.July, 30)),
	}

	// Act
	paydayExpenses := model.PutExpensesInTheirPayPeriods(payDays, expenses)

	// Assert
	list, ok := paydayExpenses["2026-06-30"]
	if !ok {
		t.Errorf("Expected 2026-06-30 to exist")
	}
	if len(list) != 3 {
		t.Errorf("len(paydayExpenses['2026-06-30']) = %d; want 3", len(list))
	}

	list2, ok2 := paydayExpenses["2026-07-15"]
	if !ok2 {
		t.Errorf("Expected 2026-07-15 to exist")
	}
	if len(list2) != 3 {
		t.Errorf("len(paydayExpenses['2026-07-15']) = %d; want 3", len(list2))
	}
}

func TestPutExpensesInTheirPayPeriods_PayDayStartsItsOwnPeriod(t *testing.T) {
	// Regression: expenses due on (or after) a pay day must fall on the pay
	// that starts on that day, not on the previous pay.
	payDays := []time.Time{
		model.Date(2026, time.July, 31),
		model.Date(2026, time.August, 15),
		model.Date(2026, time.August, 31),
		model.Date(2026, time.September, 15),
	}

	expenses := []*model.Expense{
		model.NewExpense(12, model.Date(2026, time.August, 15)), // Crave-like: on the pay day
		model.NewExpense(12, model.Date(2026, time.August, 16)),
		model.NewExpense(140, model.Date(2026, time.August, 31)), // Provision Hydro: on the pay day
		model.NewExpense(50, model.Date(2026, time.September, 1)),
	}

	paydayExpenses := model.PutExpensesInTheirPayPeriods(payDays, expenses)

	if got := len(paydayExpenses["2026-07-31"]); got != 0 {
		t.Errorf("pay starting 2026-07-31 should hold no Aug 15/16 expenses; got %d", got)
	}
	if got := len(paydayExpenses["2026-08-15"]); got != 2 {
		t.Errorf("pay starting 2026-08-15 should hold the Aug 15 and Aug 16 expenses; got %d", got)
	}
	if got := len(paydayExpenses["2026-08-31"]); got != 2 {
		t.Errorf("pay starting 2026-08-31 should hold the Aug 31 and Sep 1 expenses; got %d", got)
	}
}

func TestFilterOutPaidExpensesFromPastPays(t *testing.T) {
	// Arrange: today is 2026-08-20, so the pay starting 2026-07-31 is over,
	// the one starting 2026-08-15 is the current one.
	now := model.Date(2026, time.August, 20)
	payDays := []time.Time{
		model.Date(2026, time.July, 31),
		model.Date(2026, time.August, 15),
		model.Date(2026, time.August, 31),
	}

	paidPast := paidExpense(60, model.Date(2026, time.August, 1))
	unpaidPast := model.NewExpense(70, model.Date(2026, time.August, 2))
	paidCurrent := paidExpense(80, model.Date(2026, time.August, 16))

	grouped := model.PutExpensesInTheirPayPeriods(payDays, []*model.Expense{
		paidPast, unpaidPast, paidCurrent,
	})

	// Act
	grouped = model.FilterOutPaidExpensesFromPastPays(payDays, grouped, now)

	// Assert
	past := grouped["2026-07-31"]
	if len(past) != 1 {
		t.Fatalf("elapsed pay should only keep what is still owed; got %d expenses", len(past))
	}
	if past[0] != unpaidPast {
		t.Errorf("elapsed pay kept the wrong expense: %v", past[0])
	}

	if got := len(grouped["2026-08-15"]); got != 1 {
		t.Errorf("current pay should still show its paid expenses; got %d", got)
	}
}

func TestFilterOutPaidExpensesFromPastPays_KeepsThePayEndingToday(t *testing.T) {
	// A pay period is only over once its end date is reached: on 2026-08-15 the
	// pay starting 2026-07-31 has just ended, and the new one starts.
	now := model.Date(2026, time.August, 15)
	payDays := []time.Time{
		model.Date(2026, time.July, 31),
		model.Date(2026, time.August, 15),
	}

	grouped := model.PutExpensesInTheirPayPeriods(payDays, []*model.Expense{
		paidExpense(60, model.Date(2026, time.August, 1)),
	})

	grouped = model.FilterOutPaidExpensesFromPastPays(payDays, grouped, now)

	if got := len(grouped["2026-07-31"]); got != 0 {
		t.Errorf("pay ending today is over; its paid expenses should be hidden, got %d", got)
	}
}

// paidExpense builds an expense that is fully covered by a single payment.
func paidExpense(amount int, toBePaidAt time.Time) *model.Expense {
	id := primitive.NewObjectID()
	return model.NewExpense(amount, toBePaidAt,
		model.WithID(id),
		model.WithPayments([]model.Payment{
			{ExpenseID: id, Amount: amount, PaidAt: toBePaidAt},
		}),
	)
}

func TestPutExpensesInTheirPayPeriods_OverdueUnpaidCarriedForward(t *testing.T) {
	// Arrange
	payDays := []time.Time{
		model.Date(2026, time.June, 30),
		model.Date(2026, time.July, 15),
	}

	expID1 := primitive.NewObjectID()
	expID2 := primitive.NewObjectID()

	expenses := []*model.Expense{
		// Unpaid past expense (due 2026-06-15) -> Should be carried forward
		model.NewExpense(100, model.Date(2026, time.June, 15), model.WithID(expID1)),
		// Paid past expense (due 2026-06-10) -> Should NOT be carried forward
		model.NewExpense(50, model.Date(2026, time.June, 10), model.WithID(expID2), model.WithPayments([]model.Payment{
			{ExpenseID: expID2, Amount: 50, PaidAt: time.Now()},
		})),
	}

	// Act
	paydayExpenses := model.PutExpensesInTheirPayPeriods(payDays, expenses)

	// Assert
	list := paydayExpenses["2026-06-30"]
	if len(list) != 1 {
		t.Fatalf("Expected exactly 1 overdue expense in first pay slot, got %d", len(list))
	}
	if list[0].ID != expID1 {
		t.Errorf("Expected carried forward expense to be %v, got %v", expID1, list[0].ID)
	}
}

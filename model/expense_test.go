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
		// first pay period: (2026-06-30, 2026-07-15]
		model.NewExpense(10, model.Date(2026, time.July, 1)),
		model.NewExpense(20, model.Date(2026, time.July, 15)),
		// second pay period: (2026-07-15, 2026-07-31]
		model.NewExpense(30, model.Date(2026, time.July, 16)),
		model.NewExpense(40, model.Date(2026, time.July, 31)),
	}

	// Act
	paydayExpenses := model.PutExpensesInTheirPayPeriods(payDays, expenses)

	// Assert
	list, ok := paydayExpenses["2026-06-30"]
	if !ok {
		t.Errorf("Expected 2026-06-30 to exist")
	}
	if len(list) != 2 {
		t.Errorf("len(paydayExpenses['2026-06-30']) = %d; want 2", len(list))
	}

	list2, ok2 := paydayExpenses["2026-07-15"]
	if !ok2 {
		t.Errorf("Expected 2026-07-15 to exist")
	}
	if len(list2) != 2 {
		t.Errorf("len(paydayExpenses['2026-07-15']) = %d; want 2", len(list2))
	}
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

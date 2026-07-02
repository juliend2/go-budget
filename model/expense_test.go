package model_test

import (
	"testing"
	"time"

	"desrosiers.org/budget/model"
	"github.com/dromara/carbon/v2"
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
		// first pay:
		model.NewExpense(1, model.Date(2026, time.June, 30)),
		model.NewExpense(2, model.Date(2026, time.June, 30)),
		model.NewExpense(1, model.Date(2026, time.July, 1)),
		// second pay:
		model.NewExpense(1, model.Date(2026, time.July, 15)),
		model.NewExpense(1, model.Date(2026, time.July, 16)),
		model.NewExpense(1, model.Date(2026, time.July, 30)),
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

	list3, ok3 := paydayExpenses["2026-07-31"]
	if ok3 {
		t.Errorf("Expected 2026-07-31 NOT to exist")
	}
	if len(list3) != 0 {
		t.Errorf("len(paydayExpenses['2026-07-31']) = %d; want 0", len(list3))
	}
}

func TestIntegrationBetweenGetPayDaysAndPutExpensesInTheirPayPeriods(t *testing.T) {
	// Arrange
	dateRange := model.DateRange{
		From: model.Date(2026, time.June, 1),
		To:   model.Date(2026, time.July, 31),
	}
	payDays := model.GetPayDays(dateRange)

	expenses := []*model.Expense{
		// first pay:
		model.NewExpense(1, model.Date(2026, time.June, 30)),
		model.NewExpense(2, model.Date(2026, time.June, 30)),
		model.NewExpense(1, model.Date(2026, time.July, 1)),
		// second pay:
		model.NewExpense(1, model.Date(2026, time.July, 15)),
		model.NewExpense(1, model.Date(2026, time.July, 16)),
		model.NewExpense(1, model.Date(2026, time.July, 30)),
	}

	// Act
	paydayExpenses := model.PutExpensesInTheirPayPeriods(payDays, expenses)

	// Assert
	_, ok := paydayExpenses["2026-06-30"]
	if !ok {
		t.Errorf("2026-06-30 pay should exist")
	}

	// TODO: add more tests

}

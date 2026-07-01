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

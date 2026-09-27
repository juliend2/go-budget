package model_test

import (
	"testing"
	"time"

	"desrosiers.org/budget/model"
	"github.com/dromara/carbon/v2"
)

// Regression: nextPayDateAfter used to compute the last day of the month via
// AddMonth(), which overflows at month ends (Aug 31 + 1 month = Oct 1 because
// "Sept 31" does not exist). The whole recurring chain then jumped from Aug 31
// straight to Oct 15, silently skipping the Sept 15 and Sept 30 pay days.
func TestPayPaceDoesNotSkipPayDaysAfterEndOfMonth(t *testing.T) {
	// Arrange
	expTpl := model.NewExpenseTemplate(
		140,
		"provision",
		model.WithInitialToBePaidOn(2026, time.August, 31),
		model.WithRepeatabilityInterval(1, "P"),
	)

	// Act
	expenses, err := expTpl.GenerateRepeatingExpenses(
		model.DateRange{
			From: model.Date(2026, time.August, 15),
			To:   model.Date(2026, time.October, 31),
		},
	)

	// Assert
	if err != nil {
		t.Fatalf("GenerateRepeatingExpenses() error = %v", err)
	}
	if len(expenses) != 5 {
		t.Fatalf("len(GenerateRepeatingExpenses()) = %d; want 5", len(expenses))
	}

	want := []string{"2026-08-31", "2026-09-15", "2026-09-30", "2026-10-15", "2026-10-31"}
	for i, w := range want {
		got := carbon.NewCarbon(expenses[i].ToBePaidAt).ToDateString()
		if got != w {
			t.Errorf("expenses[%d].ToBePaidAt = %s; want %s", i, got, w)
		}
	}
}

// Regression: same overflow bug applies after Oct 31 (Nov has 30 days).
func TestPayPaceDoesNotSkipNovemberAfterOctober31(t *testing.T) {
	// Arrange
	expTpl := model.NewExpenseTemplate(
		1064,
		"loyer",
		model.WithInitialToBePaidOn(2026, time.October, 31),
		model.WithRepeatabilityInterval(1, "P"),
	)

	// Act
	expenses, err := expTpl.GenerateRepeatingExpenses(
		model.DateRange{
			From: model.Date(2026, time.October, 15),
			To:   model.Date(2026, time.December, 31),
		},
	)

	// Assert
	if err != nil {
		t.Fatalf("GenerateRepeatingExpenses() error = %v", err)
	}
	if len(expenses) != 5 {
		t.Fatalf("len(GenerateRepeatingExpenses()) = %d; want 5", len(expenses))
	}

	want := []string{"2026-10-31", "2026-11-15", "2026-11-30", "2026-12-15", "2026-12-31"}
	for i, w := range want {
		got := carbon.NewCarbon(expenses[i].ToBePaidAt).ToDateString()
		if got != w {
			t.Errorf("expenses[%d].ToBePaidAt = %s; want %s", i, got, w)
		}
	}
}

// The generated pay-pace chain must never produce a date that is neither the
// 15th nor the last day of its month, and never skip a pay day.
func TestPayPaceChainIsContinuous(t *testing.T) {
	// Arrange
	expTpl := model.NewExpenseTemplate(
		50,
		"café",
		model.WithInitialToBePaidOn(2026, time.January, 15),
		model.WithRepeatabilityInterval(1, "P"),
	)

	// Act
	expenses, err := expTpl.GenerateRepeatingExpenses(
		model.DateRange{
			From: model.Date(2026, time.January, 15),
			To:   model.Date(2027, time.March, 31),
		},
	)

	// Assert
	if err != nil {
		t.Fatalf("GenerateRepeatingExpenses() error = %v", err)
	}

	for i := 1; i < len(expenses); i++ {
		prev := carbon.NewCarbon(expenses[i-1].ToBePaidAt)
		cur := carbon.NewCarbon(expenses[i].ToBePaidAt)

		// every occurrence is on the 15th or the last day of its month
		if cur.Day() != 15 && cur.Day() != cur.EndOfMonth().Day() {
			t.Errorf("occurrence %d = %s; want the 15th or last day of month", i, cur.ToDateString())
		}
		// occurrences advance strictly by one pay period
		expectedNext := nextExpectedPayDayAfter(prev.StdTime())
		if cur.ToDateString() != expectedNext {
			t.Errorf("occurrence %d = %s; want %s (chain skipped a pay day)", i, cur.ToDateString(), expectedNext)
		}
	}
}

// nextExpectedPayDayAfter is an independent reimplementation of the pay-day
// rule (15th or last day of month) using only the standard library, so a bug
// in model.date cannot hide behind itself.
func nextExpectedPayDayAfter(t time.Time) string {
	y, m, d := t.Date()
	if d < 15 {
		return time.Date(y, m, 15, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	}
	firstOfNext := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
	lastOfThis := firstOfNext.AddDate(0, 0, -1)
	if d < lastOfThis.Day() {
		return lastOfThis.Format("2006-01-02")
	}
	return time.Date(y, m, 15, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0).Format("2006-01-02")
}
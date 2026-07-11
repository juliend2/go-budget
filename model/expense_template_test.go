package model_test

import (
	"testing"
	"time"

	"desrosiers.org/budget/model"
	"github.com/dromara/carbon/v2"
)

func TestExpenseTemplateRepetitionCreatesExpenseForTemplateStartingBeforeFrom(t *testing.T) {
	// Arrange
	expTpl := model.NewExpenseTemplate(
		100,
		"abonnement",
		model.WithInitialToBePaidOn(2026, time.May, 1),
		model.WithRepeatabilityInterval(1, "M"),
	)

	// Act
	expenses, _ := expTpl.GenerateRepeatingExpenses(
		model.DateRange{
			From: model.Date(2026, time.June, 1),
			To:   model.Date(2026, time.June, 2), // inclusive
		},
	)

	// Assert
	if len(expenses) != 2 {
		t.Errorf("len(GenerateRepeatingExpenses()) = %d; want 2", len(expenses))
	}

	// Verify metadata copy
	if expenses[0].Description != "abonnement" {
		t.Errorf("expenses[0].Description = %s; want abonnement", expenses[0].Description)
	}
	if expenses[0].TemplateID == nil || *expenses[0].TemplateID != expTpl.ID {
		t.Errorf("expenses[0].TemplateID = %v; want %v", expenses[0].TemplateID, expTpl.ID)
	}
}

func TestExpenseTemplateRepetitionCreatesFirstTwoOccurrences(t *testing.T) {
	// Arrange
	expTpl := model.NewExpenseTemplate(
		100,
		"abonnement",
		model.WithInitialToBePaidOn(2026, time.June, 1),
		model.WithRepeatabilityInterval(1, "M"),
	)

	// Act
	expenses, _ := expTpl.GenerateRepeatingExpenses(
		model.DateRange{
			From: model.Date(2026, time.May, 29),
			To:   model.Date(2026, time.July, 29),
		},
	)

	// Assert
	if len(expenses) != 2 {
		t.Errorf("len(GenerateRepeatingExpenses()) = %d; want 2", len(expenses))
	}
}

func TestExpenseCreationOfFirstExpenseRepetitionThatIsAfterDateRangeStart(t *testing.T) {
	// Arrange
	expTpl := model.NewExpenseTemplate(
		100,
		"abonnement",
		model.WithInitialToBePaidOn(2026, time.June, 1),
		model.WithRepeatabilityInterval(1, "M"),
	)

	dateRange := model.DateRange{
		From: model.Date(2026, time.May, 31),  // matches first pay day
		To:   model.Date(2026, time.July, 31), // matches last pay day
	}

	expenses, _ := expTpl.GenerateRepeatingExpenses(dateRange)
	exp := expenses[0]
	if carbon.NewCarbon(exp.ToBePaidAt).ToDateString() != "2026-06-01" {
		t.Errorf("exp.ToBePaidAt = %s; want 2026-06-01", carbon.NewCarbon(exp.ToBePaidAt).ToDateString())
	}
}

func TestIntegrationRepeatingExpensesAreProperlyFilledIntoPayDayGroupings(t *testing.T) {
	// Arrange
	expTpl := model.NewExpenseTemplate(
		100,
		"abonnement",
		model.WithInitialToBePaidOn(2026, time.June, 1),
		model.WithRepeatabilityInterval(1, "M"),
	)

	dateRange := model.DateRange{
		From: model.Date(2026, time.May, 31),  // matches first pay day
		To:   model.Date(2026, time.July, 31), // matches last pay day
	}

	expenses, _ := expTpl.GenerateRepeatingExpenses(dateRange)
	payDays := model.GetPayDays(dateRange)

	// Act
	paydayExpenses := model.PutExpensesInTheirPayPeriods(payDays, expenses)

	// Assert
	_, ok := paydayExpenses["2026-05-31"]
	if !ok {
		t.Errorf("2026-05-31 pay should exist") // ... even if the only expense is a tpl that starts after the PayDay
	}

	firstMonthExpenses, _ := paydayExpenses["2026-05-31"]
	if len(firstMonthExpenses) < 1 {
		t.Errorf("2026-05-31 pay should contain the first repeating expense")
	}
}

func TestInvalidPaceReturnsError(t *testing.T) {
	// Arrange
	expTpl := model.NewExpenseTemplate(
		100,
		"abonnement",
		model.WithInitialToBePaidOn(2026, time.June, 1),
		model.WithRepeatabilityInterval(1, "INVALID"),
	)

	// Act
	_, err := expTpl.GenerateRepeatingExpenses(
		model.DateRange{
			From: model.Date(2026, time.June, 1),
			To:   model.Date(2026, time.July, 1),
		},
	)

	// Assert
	if err == nil {
		t.Errorf("Expected error for invalid repeatability interval pace, got nil")
	}
}

func TestInvalidUnitReturnsError(t *testing.T) {
	// Arrange
	expTpl := model.NewExpenseTemplate(
		100,
		"abonnement",
		model.WithInitialToBePaidOn(2026, time.June, 1),
		model.WithRepeatabilityInterval(0, "M"),
	)

	// Act
	_, err := expTpl.GenerateRepeatingExpenses(
		model.DateRange{
			From: model.Date(2026, time.June, 1),
			To:   model.Date(2026, time.July, 1),
		},
	)

	// Assert
	if err == nil {
		t.Errorf("Expected error for interval unit <= 0, got nil")
	}
}

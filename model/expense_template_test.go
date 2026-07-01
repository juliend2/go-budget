package model_test

import (
	"testing"
	"time"

	"desrosiers.org/budget/model"
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
			To:   model.Date(2026, time.June, 2), // excludes the `To`
		},
	)

	// Assert
	if len(expenses) != 2 {
		t.Errorf("len(GenerateRepeatingExpenses()) = %d; want 2", len(expenses))
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

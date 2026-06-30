package repository_test

import (
	"testing"
	"time"

	"desrosiers.org/budget/model"
	"desrosiers.org/budget/repository"
)

func TestExpenseTemplateRepetitionCreatesFirstTwoOccurrences(t *testing.T) {
	// Arrange
	expTpl := model.NewExpenseTemplate(
		100,
		"abonnement",
		model.WithInitialToBePaidOn(2026, time.June, 1),
		model.WithRepeatabilityInterval(1, "m"),
	)

	// Act
	expenses := repository.GenerateRepeatingExpenses(
		expTpl,
		model.DateRange{
			From: model.Date(2026, time.June, 1),
			To:   model.Date(2026, time.July, 29),
		},
	)

	// Assert
	if len(expenses) != 2 {
		t.Errorf("len(GenerateRepeatingExpenses()) = %d; want 2", len(expenses))
	}
}

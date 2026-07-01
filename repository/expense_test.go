package repository_test

import (
	"testing"
	"time"

	"desrosiers.org/budget/model"
	"desrosiers.org/budget/repository"
)

func TestFillCandidates(t *testing.T) {
	// Arrange
	dtRange := model.DateRange{
		From: time.Date(2026, time.June, 10, 01, 0, 0, 0, time.UTC),
		To:   time.Date(2026, time.July, 10, 01, 0, 0, 0, time.UTC),
	}
	expenseTemplates := []*model.ExpenseTemplate{
		model.NewExpenseTemplate(100, "exp", // should be considered since it starts during the month
			model.WithInitialToBePaidOn(2026, time.June, 11),
			model.WithRepeatabilityInterval(1, "m"),
		),
		model.NewExpenseTemplate(110, "exp2", // shouldn't be considered since it starts in the future
			model.WithInitialToBePaidOn(2027, time.October, 31),
			model.WithRepeatabilityInterval(2, "m"),
		),
	}
	existRecords := []*model.Expense{}

	// Act
	expenses := repository.FillCandidates(expenseTemplates, dtRange, existRecords)

	// Assert
	if len(expenses) == 0 {
		t.Errorf("len(FillCandidates()) = %d; want 1", len(expenses))
	}
}

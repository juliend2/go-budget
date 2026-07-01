package repository

import (
	"time"

	"desrosiers.org/budget/model"
)

// type Expense struct{}

func FillCandidates(expTpls []*model.ExpenseTemplate, dtRange model.DateRange, existingExp []*model.Expense) []*model.Expense {
	// TODO

	return []*model.Expense{
		model.NewExpense(100, time.Date(2026, time.June, 11, 0, 0, 0, 0, time.UTC)),
	}
}

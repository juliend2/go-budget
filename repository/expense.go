package repository

import (
	"time"

	"desrosiers.org/budget/model"
)

// type Expense struct{}

func FillCandidates(dtRange model.DateRange, expTpls []*model.ExpenseTemplate, existingExp []*model.Expense) []*model.Expense {
	// TODO

	return []*model.Expense{
		model.NewExpense(100, time.Date(2026, time.June, 11, 0, 0, 0, 0, time.UTC)),
	}
}

package model_test

import (
	"fmt"
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
			To:   model.Date(2026, time.June, 2), // inclusive
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

	dateRange := model.DateRange{
		From: model.Date(2026, time.May, 31),  // matches first pay day
		To:   model.Date(2026, time.July, 31), // matches last pay day
	}

	expenses, _ := expTpl.GenerateRepeatingExpenses(dateRange)
	payDays := model.GetPayDays(dateRange)

	// Act
	paydayExpenses := model.PutExpensesInTheirPayPeriods(payDays, expenses)

	for key, val := range paydayExpenses {
		fmt.Println(key)
		for _, exp := range val {
			fmt.Println(exp)
		}
	}
	// Assert
	_, ok := paydayExpenses["2026-05-31"]
	if !ok {
		t.Errorf("2026-05-31 pay should exist") // ... even if the only expense is a tpl that starts after the PayDay
	}

	firstMonthExpenses, _ := paydayExpenses["2026-05-01"]
	if len(firstMonthExpenses) < 1 {
		t.Errorf("2026-05-01 pay should contain the first repeating expense")
	}
}

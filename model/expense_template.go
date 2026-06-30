package model

import "time"

type ExpenseTemplate struct {
	Id                        int
	Amount                    int
	Description               string
	InitialToBePaidOn         time.Time
	RepeatabilityIntervalUnit int
	RepeatabilityIntervalPace string
	IsOnHold                  bool
}

type ExpenseTemplateOption func(*ExpenseTemplate)

func NewExpenseTemplate(amnt int, desc string, opt ...ExpenseTemplateOption) *ExpenseTemplate {
	expenseTemplate := &ExpenseTemplate{
		Amount:      amnt,
		Description: desc,
	}

	for _, o := range opt {
		o(expenseTemplate)
	}

	return expenseTemplate
}

func WithInitialToBePaidOn(y int, m time.Month, d int) ExpenseTemplateOption {
	return func(e *ExpenseTemplate) {
		e.InitialToBePaidOn = time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	}
}

func WithRepeatabilityInterval(unit int, pace string) ExpenseTemplateOption {
	return func(e *ExpenseTemplate) {
		e.RepeatabilityIntervalUnit = unit
		e.RepeatabilityIntervalPace = pace
	}
}

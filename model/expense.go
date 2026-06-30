package model

import (
	"time"
)

type DateRange struct {
	From time.Time
	To   time.Time
}

type Expense struct {
	now        time.Time // DI for testing
	Amount     int       // dollars
	ToBePaidAt time.Time
}

type ExpenseOption func(*Expense)

func NewExpense(amount int, toBePaidAt time.Time, opt ...ExpenseOption) *Expense {
	expense := &Expense{
		now:        time.Now(),
		Amount:     amount,
		ToBePaidAt: toBePaidAt,
	}

	for _, o := range opt {
		o(expense)
	}

	return expense
}

func WithNow(now time.Time) ExpenseOption {
	return func(e *Expense) {
		e.now = now
	}
}

func (e *Expense) IsDue() bool {
	return !e.now.Before(e.ToBePaidAt)
}

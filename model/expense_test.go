package model_test

import (
	"testing"
	"time"

	"desrosiers.org/budget/model"
)

func TestIsDue(t *testing.T) {
	exp := model.NewExpense(
		50,
		time.Now().Add(time.Hour*24*10),
		model.WithNow(time.Now().Add(time.Hour*24*1)),
	)
	got := exp.IsDue()
	if got == true {
		t.Errorf("IsDue() = %v; want false", got)
	}
}

README
======

```
# First step: Get the expenses:

dateRange := INPUT[0]
exp_templates := repo.GetExpenseTemplates() // O(N)
templated_expense_ids := []

for each exp_template from exp_templates:
    repeating_expenses_in_range := model.GenerateRepeatingExpenses(exp_template, dateRange) // 
    for each repeating_exp from repeating_expenses_in_range:
        exp_id := repo.GetOrInsertExpense(repeating_exp) // returns the full expense, including the payments (if any). NOT using upsert, since we want the id if it exists
        templated_expense_ids.push(exp_id)

expenses_without_template := repo.FillExpensesFromIds(repo.GetExpensesIdsWithoutTemplate(dateRange))
templated_expenses := repo.FillExpensesFromIds(templated_expense_ids)

expenses := MergeExpenses(templated_expenses, expenses_without_template)

# Next step is to split those expenses in half-months (pays)

## generate a []time.Time of pay days

first_expense := expenses[0]
pay_days := [first_expense]
last_expense := expenses[:1]

for ;pay_days.ToBePaidAt < last_expense.ToBePaidAt; {
    current_pay_day := pay_days[:1]
    var next_pay_day
    if current_pay_day.ToBePaidAt.Day() == 15:
        next_pay_day = get_next_first_of_month_from(current_pay_day)
    else:
        next_pay_day = get_next_15th_from(current_pay_day)

    pay_days.push(next_pay_day)
}

## Add the expenses to the dict inside the proper pay days:

accumulator := {}
for i from pay_days:
    from := pay_days[i]
    to := pay_days[i+1]

    for each expense from expenses:
        if expense.ToBePaidAt >= from && (is_null(to) || expense.ToBePaidAt < to):
            if accumulator[pay_day]:
                accumulator[pay_day] = append(accumulator[pay_day], expense)
            else:
                accumulator[pay_day] = [expense]
```


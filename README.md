README
======

```
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

```

    // passe les expense templates ainsi que le date range
    (expense)           func GenerateRepeatingExpenses(expTpl *model.ExpenseTemplate, dateRange model.DateRange) ([]*model.Expense, error)

    // ca genere les expenses qui existent dans la DB

    // 
    (expense_template)  func FillCandidates(expTpls []*model.ExpenseTemplate, dtRange model.DateRange, existingExp []*model.Expense) []*model.Expense



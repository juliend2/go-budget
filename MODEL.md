# Budget Application: Business Logic Assessment & Aligned Architecture

This document tracks the assessment of the Go business logic and the decisions aligned on with the user.

---

## 📅 1. Payday Schedule & Period Grouping

### Aligned Rules
You receive a paycheck twice a month: on the **15th** and on the **last day of the month**. 
* **Pay received on the Last Day of Month $M-1$** pays for expenses due **Day 1 to Day 15 of Month $M$** (Summary date: `M/15`).
* **Pay received on the 15th of Month $M$** pays for expenses due **Day 16 to Last Day of Month $M$** (Summary date: `M/LastDay`).

Thus, the pay periods are structured as `(from, to]` half-month intervals:
* **Period 1**: `07/01/2026` to `07/15/2026` inclusive (Summary: `07/15/2026`).
* **Period 2**: `07/16/2026` to `07/31/2026` inclusive (Summary: `07/31/2026`).
* **Period 3**: `08/01/2026` to `08/15/2026` inclusive (Summary: `08/15/2026`).

### Overdue/Unpaid Expenses
* **Behavior**: Unpaid expenses from previous periods (where `ToBePaidAt` is prior to the start of the first displayed period) must be carried forward and grouped into the **first visible pay period**.
* **Total Calculation**: Their amounts will be accumulated in the first payday's summary (`Sommaire en date du`).
* **Implementation**: We will refactor `PutExpensesInTheirPayPeriods()` to filter and group these outstanding past expenses accordingly.

---

## 🔄 2. Repeating Expense Generation

* **Infinite Loop Bug**: **To be fixed.** We will validate the pace inside `GenerateRepeatingExpenses()` and handle/propagate errors returned by `getNextToBePaidAt()`.
* **Future Expense Generation Leak**: No change required. We will leave the logic as is.
* **Performance / Efficiency**: No change required for now. We will wait until performance issues become perceptible before optimizing the regeneration logic.

---

## 🗃️ 3. Aligned Data Model Schema

We will extend the domain models to support MongoDB persistence and the HTML view:

### `Payment` Model (New)
Tracks payments made toward individual expenses. An expense can be paid in full or across multiple payments.
* `ID`: ObjectID
* `ExpenseID`: ObjectID
* `Amount`: int (dollars)
* `PaidAt`: time.Time

### `Expense` Model (Extended)
* `ID`: ObjectID
* `Description`: string (e.g. "Rent", "Netflix")
* `Amount`: int (dollars)
* `ToBePaidAt`: time.Time
* `TemplateID`: *ObjectID (nullable; nil for one-time expenses)
* `Payments`: []Payment (slice loaded at runtime)
* **Calculated Paid Status**: We will add an `IsPaid()` method. If the sum of all associated `Payments` is `>= Expense.Amount`, the expense is considered paid.

### `ExpenseTemplate` Model
* `ID`: ObjectID
* `Description`: string
* `Amount`: int
* `InitialToBePaidOn`: time.Time
* `RepeatabilityIntervalUnit`: int
* `RepeatabilityIntervalPace`: string
* `IsOnHold`: bool

---

## 🐳 4. Database & Deployment Strategy

* **Database**: MongoDB.
* **Deployment**: Docker Compose (`docker-compose.yaml`) for local development, designed to be easily deployed to production.

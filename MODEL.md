# Budget Application: Business Logic Assessment & Aligned Architecture

This document tracks the assessment of the Go business logic and the decisions aligned on with the user.

---

## 📅 1. Payday Schedule & Period Grouping

### Aligned Rules
You receive a paycheck twice a month: on the **15th** and on the **last day of the month**. Each pay period is dated by the day the money is **received** (the start of the slot), because that pay covers every expense in the slot.
* **Pay received on the Last Day of Month $M-1$** pays for expenses due **Last Day of $M-1$ to Day 14 of Month $M$** (Summary date: the pay day, `(M-1)/LastDay`).
* **Pay received on the 15th of Month $M$** pays for expenses due **Day 15 to before-last day of Month $M$** (Summary date: the pay day, `M/15`).

Thus, the pay periods are structured as `[from, to)` half-month intervals: a pay day **starts** its own period, so an expense due *on* a pay day belongs to that pay day's period (not the previous one). The summary (`En date du`) is labeled with the period's start — the day the pay is received.
* **Period 1**: `06/30/2026` (inclusive) to `07/15/2026` (exclusive) (Summary: `06/30/2026`).
* **Period 2**: `07/15/2026` (inclusive) to `07/31/2026` (exclusive) (Summary: `07/15/2026`).
* **Period 3**: `07/31/2026` (inclusive) to `08/15/2026` (exclusive) (Summary: `07/31/2026`).

### Overdue/Unpaid Expenses
* **Behavior**: Unpaid expenses from previous periods (where `ToBePaidAt` is prior to the start of the first displayed period) must be carried forward and grouped into the **first visible pay period**.
* **Total Calculation**: Their amounts will be accumulated in the first payday's summary (`Sommaire en date du`).


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
* `RepeatabilityIntervalPace`: string — `D` (day), `W` (week), `M` (month), `Y` (year), or `P` (pay: the 15th and last day of the month; the unit is the number of pay days to skip, e.g. 1 = every pay, 2 = every other pay)
* `IsOnHold`: bool

---

## 🐳 4. Database & Deployment Strategy

* **Database**: MongoDB.
* **Deployment**: Docker Compose (`docker-compose.yaml`) for local development, designed to be easily deployed to production.

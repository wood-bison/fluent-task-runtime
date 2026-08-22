# Fluent calculator

Implement `createCalculator(initialValue = 0)` in `calculator.js`.

The function returns a small stateful calculator with these members:

- `.add(value)`
- `.subtract(value)`
- `.multiply(value)`
- `.div(value)`
- `.value` — the current number

Every operation mutates this calculator and returns the same calculator so a
learner can write `createCalculator(10).add(5).div(3).value`.

Division by zero is an error boundary: `.div(0)` must throw an error whose
message mentions zero, and the value must be unchanged. Do not use module-level
state — two calculators created side by side must not affect each other.

Start with a chain you can reason about:

```js
const result = createCalculator(10)
  .add(5)
  .div(3)
  .value; // 5
```

import test from 'node:test';
import assert from 'node:assert/strict';
import { createCalculator } from '../calculator.js';

test('starts at the initial value', () => {
  assert.equal(createCalculator(7).value, 7);
  assert.equal(createCalculator().value, 0);
});

test('adds and subtracts without losing state', () => {
  const calculator = createCalculator(10);
  assert.strictEqual(calculator.add(5), calculator);
  assert.strictEqual(calculator.subtract(3), calculator);
  assert.equal(calculator.value, 12);
});

test('multiplies and divides in a fluent chain', () => {
  assert.equal(createCalculator(6).multiply(4).div(3).value, 8);
});

test('rejects division by zero without mutating state', () => {
  const calculator = createCalculator(9);
  assert.throws(() => calculator.div(0), /zero/u);
  assert.equal(calculator.value, 9);
});

test('returns the current value through the value getter', () => {
  const calculator = createCalculator(2);
  calculator.add(3);
  assert.equal(calculator.value, 5);
});

test('keeps independent calculator instances independent', () => {
  const first = createCalculator(1);
  const second = createCalculator(1);
  first.add(9);
  assert.equal(first.value, 10);
  assert.equal(second.value, 1);
});

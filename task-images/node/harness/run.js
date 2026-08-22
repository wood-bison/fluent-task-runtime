'use strict';

// The Exercism test-runner interface (ADR-0006), for Node's own `node:test`.
//
// Contract: read a read-only learner workspace and a separate read-only hidden
// test mount, run the tests, and write results.json to a writable output
// directory. Hidden tests are deliberately not copied into /solution: that
// path is the learner's only visible workspace.
// Exit code 0 means THIS SCRIPT completed, never that the tests passed — a
// mixed or all-failing test run still exits 0 and reports its verdict inside
// results.json. Only a genuine harness malfunction (an exception thrown here,
// outside the try/catch below) may exit non-zero.

const fs = require('node:fs');
const path = require('node:path');
const { run } = require('node:test');

// The mount points container.executor always uses. Overridable only so this
// harness can be exercised outside a container (the test suite does); the
// image itself never sets these, so a real run always sees the defaults.
const INPUT_DIR = process.env['HARNESS_INPUT_DIR'] || '/solution';
const TESTS_DIR = process.env['HARNESS_TESTS_DIR'] || '/hidden-tests/tests';
const OUTPUT_DIR = process.env['HARNESS_OUTPUT_DIR'] || '/output';
const OUTPUT_FILE = path.join(OUTPUT_DIR, 'results.json');

const RESULTS_VERSION = 2;

function findTestFiles(dir) {
  if (!fs.existsSync(dir)) {
    return [];
  }
  const found = [];
  for (const entry of fs.readdirSync(dir, { recursive: true })) {
    if (/\.test\.m?js$/u.test(entry)) {
      found.push(path.join(dir, entry));
    }
  }
  return found;
}

// Classifies a failed test's node:test error shape into our two failure
// kinds: an assertion that ran and did not hold ('fail') versus anything
// else the test body threw ('error') — a TypeError, a rejected promise the
// test did not expect, and so on. Both are legal individual-test statuses
// under a document whose overall status is "fail" (only "error" and "pass"
// documents require every test to share their status).
function classifyFailure(detail) {
  const error = detail && detail.error;
  const cause = error && error.cause;

  if (cause && typeof cause === 'object' && cause.code === 'ERR_ASSERTION') {
    const expected = safeStringify(cause.expected);
    const actual = safeStringify(cause.actual);
    const operator = typeof cause.operator === 'string' ? cause.operator : 'assertion';
    return {
      status: 'fail',
      message: `${operator}: expected ${expected}, got ${actual}`,
    };
  }

  if (typeof cause === 'string') {
    return { status: 'error', message: cause };
  }

  if (cause && typeof cause.message === 'string') {
    return { status: 'error', message: cause.message };
  }

  if (error && typeof error.message === 'string') {
    return { status: 'error', message: error.message };
  }

  return { status: 'error', message: 'The test threw without a recognisable message.' };
}

function safeStringify(value) {
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

async function main() {
  const testFiles = findTestFiles(TESTS_DIR);

  if (testFiles.length === 0) {
    // This is OUR packaging mistake (a task authored without `tests/`, or a
    // test file whose name misses `.test.m?js$`, or a mount that silently
    // failed) — never the learner's. A `status: 'error'` document means
    // "the learner's code did not parse", so writing one here would tell
    // the learner their code is broken when it is ours. Throwing instead
    // makes `main()`'s own `.catch` below exit non-zero, which
    // `container.executor.ts` already reports as a harness failure, not a
    // graded attempt.
    throw new Error(`No hidden test files found under ${TESTS_DIR}.`);
  }

  const tests = [];
  // Per file: did the file itself register any real test, or did loading it
  // fail outright (a syntax error, a throw at module scope)? node:test
  // reports a load failure as a single test:fail event whose name equals the
  // file path — no real test names exist to compare against declaredTests.
  const loadFailedFiles = new Set(testFiles);
  const stderrByFile = new Map();

  const stream = run({ files: testFiles, concurrency: 1 });

  for await (const event of stream) {
    if (event.type === 'test:stderr') {
      const key = event.data.file;
      const existing = stderrByFile.get(key) ?? '';
      stderrByFile.set(key, existing + event.data.message);
      continue;
    }

    if (event.type !== 'test:pass' && event.type !== 'test:fail') {
      continue;
    }

    const { data } = event;
    const isFileWrapper = data.name === data.file;
    if (isFileWrapper) {
      // The file produced at least this event; whether it "loaded" is
      // decided below by whether any REAL test was also seen for it.
      continue;
    }

    loadFailedFiles.delete(data.file);

    if (event.type === 'test:pass') {
      tests.push({ name: data.name, status: 'pass' });
    } else {
      const { status, message } = classifyFailure(data.details);
      tests.push({ name: data.name, status, message });
    }
  }

  if (tests.length === 0) {
    // Every test file failed before registering a single test — a compile
    // or load failure. The learner's code did not run at all.
    const message =
      [...stderrByFile.values()].join('').trim() ||
      'The submitted file could not be loaded.';
    writeResults({
      version: RESULTS_VERSION,
      status: 'error',
      message: truncate(message, 65_535),
    });
    return;
  }

  const allPassed = tests.every((test) => test.status === 'pass');
  writeResults({
    version: RESULTS_VERSION,
    status: allPassed ? 'pass' : 'fail',
    tests,
  });
}

function truncate(value, limit) {
  return value.length > limit ? value.slice(0, limit) : value;
}

function writeResults(document) {
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });
  fs.writeFileSync(OUTPUT_FILE, JSON.stringify(document, null, 2), 'utf8');
}

main().catch((error) => {
  // A genuine harness malfunction — not a learner test failure. Print for
  // the container's captured stdout/stderr and exit non-zero, which
  // container.executor reports as a harness failure rather than a verdict.
  console.error('[harness] fatal error:', error);
  process.exitCode = 1;
});

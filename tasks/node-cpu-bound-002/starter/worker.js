import { parentPort, workerData } from 'node:worker_threads';

const end = Date.now() + Math.max(0, workerData.durationMs ?? 5);
let accumulator = 0;
while (Date.now() < end) {
  accumulator = (accumulator + 1) % 1_000_003;
}

parentPort.postMessage({ result: workerData.value * workerData.value, accumulator });

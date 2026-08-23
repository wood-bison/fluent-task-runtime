emit('start', 'sync', 'script-start');
schedule('timer-scheduled', 'Timer scheduled', 'setTimeout(callback, 0)', 'timers');
setTimeout(() => emit('timeout', 'timers', 'timer-run'), 0);
schedule('promise-scheduled', 'Promise reaction queued', 'Promise.then(callback)', 'microtask');
Promise.resolve().then(() => emit('promise', 'microtask', 'promise-run'));
schedule('next-tick-scheduled', 'nextTick queued', 'process.nextTick(callback)', 'nextTick');
process.nextTick(() => emit('nextTick', 'nextTick', 'next-tick-run', 'console.log("nextTick")', 'nextTick queue drained'));
emit('end', 'sync', 'sync-end');

# Deferred

Sometimes you need a promise whose settlement happens somewhere else, not
inside the executor function you pass to `new Promise(...)`. Code that
listens for an event, waits on a callback-based API, or hands a promise to
one part of a system while settling it from another needs a way to create
the promise first and settle it later, from the outside.

A `Deferred` is that object. Constructing one immediately gives you three
things: a `.promise` property holding a promise in the pending state, a
`.resolve(value)` method, and a `.reject(reason)` method. Anyone holding the
`Deferred` can call `.then` or `.catch` on its `.promise` exactly as they
would on any promise, and anyone holding the `Deferred` can call `.resolve`
or `.reject` whenever they're ready to decide what the outcome should be.

Your task is to implement `Deferred` so it behaves exactly like a native
promise once settled, with seven behaviours checked by the hidden tests:

A freshly constructed `Deferred` starts pending — nothing has happened yet,
and no callback should fire until it is settled.

Calling the resolve function with a value settles the promise with that
value, and every `then` callback attached to it receives that value.

Calling the reject function with a reason settles the promise with that
rejection, and every `catch` callback attached to it receives that reason.

If the value passed to resolve is itself a thenable — something with its
own `then` method, such as another promise — the `Deferred` must wait for
that thenable to settle and adopt its outcome, the same way native promise
resolution does.

The same adoption applies one step downstream: if a callback passed to
`then` returns a thenable, the promise `then` returns must wait for that
thenable to settle and adopt its outcome too, rather than resolving with
the thenable object itself.

An error thrown inside a `then` callback must propagate to the next
`catch` in the chain, just as it would for any other promise.

Once a `Deferred` has been settled, settling it again — through either
function, with any value or reason — must have no further effect. The
first settlement wins and every later call is silently ignored.

The export shape in `deferred.js` is fixed: the tests import the `Deferred`
class exactly as it is declared in the starter file, and construct it with
no arguments. The contract is fixed too: every instance must expose
`.promise`, `.resolve(value)`, and `.reject(reason)` under exactly those
names. Do not rename the class, any of the three members, or change how the
class is exported. Everything else about the implementation is yours to
decide.

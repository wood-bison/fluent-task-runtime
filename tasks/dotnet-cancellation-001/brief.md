# .NET cancellation boundary

Implement `CancellationDemo.RunAsync(CancellationToken cancellationToken)` in
`cancellation.cs`.

The method represents a small piece of request work: it should asynchronously
wait for the work to complete, return `42` on a normal token, and observe the
token while waiting. If the caller cancels before completion, the returned
task must end with `OperationCanceledException` (or an equivalent cancelled
task), not continue in the background and return a value later.

The runner compiles your file in an official .NET SDK image and executes both a
normal request and a cancelled request. There are no NuGet dependencies in this
task.

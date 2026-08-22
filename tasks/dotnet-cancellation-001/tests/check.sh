#!/usr/bin/env bash
set -Eeuo pipefail

mkdir -p /tmp/dotnet-eval /output
export DOTNET_CLI_HOME=/tmp/dotnet-home
export NUGET_PACKAGES=/tmp/nuget
mkdir -p "$DOTNET_CLI_HOME" "$NUGET_PACKAGES"

cat >/tmp/dotnet-eval/Eval.csproj <<'EOF'
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
  </PropertyGroup>
  <ItemGroup>
    <Compile Include="/solution/cancellation.cs" />
  </ItemGroup>
</Project>
EOF

cat >/tmp/dotnet-eval/Program.cs <<'EOF'
using System.Text.Json;
using System.Threading;

var tests = new List<object>();

try
{
    var value = await CancellationDemo.RunAsync(CancellationToken.None);
    tests.Add(new { name = "returns the completed value", status = value == 42 ? "pass" : "fail", message = value == 42 ? (string?)null : $"expected 42, got {value}" });
}
catch (Exception error)
{
    tests.Add(new { name = "returns the completed value", status = "error", message = error.Message });
}

try
{
    using var source = new CancellationTokenSource();
    var pending = CancellationDemo.RunAsync(source.Token);
    source.CancelAfter(25);
    await pending;
    tests.Add(new { name = "honours cancellation before completion", status = "fail", message = "the task completed after its token was cancelled" });
}
catch (OperationCanceledException)
{
    tests.Add(new { name = "honours cancellation before completion", status = "pass", message = (string?)null });
}
catch (Exception error)
{
    tests.Add(new { name = "honours cancellation before completion", status = "error", message = error.Message });
}

var passed = tests.All(test => (string)test.GetType().GetProperty("status")!.GetValue(test)! == "pass");
var document = new { version = 2, status = passed ? "pass" : "fail", tests };
await File.WriteAllTextAsync("/output/results.json", JsonSerializer.Serialize(document));
EOF

set +e
dotnet restore /tmp/dotnet-eval/Eval.csproj --ignore-failed-sources >/tmp/dotnet-restore.log 2>&1
restore_status=$?
if [[ "$restore_status" == "0" ]]; then
  dotnet build /tmp/dotnet-eval/Eval.csproj --no-restore >/tmp/dotnet-build.log 2>&1
  build_status=$?
  if [[ "$build_status" == "0" ]]; then
    dotnet /tmp/dotnet-eval/bin/Debug/net10.0/Eval.dll >/tmp/dotnet-run.log 2>&1
    run_status=$?
  else
    run_status=$build_status
  fi
else
  run_status=$restore_status
fi
set -e

if [[ "$run_status" != "0" ]]; then
  cat /tmp/dotnet-restore.log /tmp/dotnet-build.log /tmp/dotnet-run.log >&2 || true
  printf '{"version":2,"status":"error","message":"The submitted C# did not compile or run."}\n' >/output/results.json
fi

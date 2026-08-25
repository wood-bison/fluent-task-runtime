#!/usr/bin/env bash
set -Eeuo pipefail

mkdir -p /tmp/dotnet-rate-limiter-eval /output
export DOTNET_CLI_HOME=/tmp/dotnet-rate-limiter-home
export NUGET_PACKAGES=/tmp/dotnet-rate-limiter-nuget
mkdir -p "$DOTNET_CLI_HOME" "$NUGET_PACKAGES"

cat >/tmp/dotnet-rate-limiter-eval/Eval.csproj <<'EOF'
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net10.0</TargetFramework>
    <ImplicitUsings>enable</ImplicitUsings>
    <Nullable>enable</Nullable>
  </PropertyGroup>
  <ItemGroup>
    <Compile Include="/solution/RateLimiter.cs" />
  </ItemGroup>
</Project>
EOF

cat >/tmp/dotnet-rate-limiter-eval/Program.cs <<'EOF'
using System;
using System.Text.Json;

var tests = new List<object>();
var epoch = DateTimeOffset.UnixEpoch;

void Check(string name, Func<bool> assertion)
{
    try
    {
        var passed = assertion();
        tests.Add(new { name, status = passed ? "pass" : "fail", message = passed ? (string?)null : "assertion returned false" });
    }
    catch (Exception error)
    {
        tests.Add(new { name, status = "error", message = error.Message });
    }
}

Check("allows requests inside the configured burst", () =>
{
    var limiter = new RateLimiter(2, 1);
    return limiter.Allow("a", epoch) && limiter.Allow("a", epoch) && !limiter.Allow("a", epoch);
});

Check("refills tokens over elapsed time", () =>
{
    var limiter = new RateLimiter(1, 1);
    return limiter.Allow("a", epoch) && !limiter.Allow("a", epoch) && limiter.Allow("a", epoch.AddSeconds(1));
});

Check("isolates independent keys", () =>
{
    var limiter = new RateLimiter(1, 1);
    return limiter.Allow("a", epoch) && limiter.Allow("b", epoch);
});

Check("rejects requests after the bucket is empty", () =>
{
    var limiter = new RateLimiter(1, 0);
    return limiter.Allow("a", epoch) && !limiter.Allow("a", epoch.AddHours(1));
});

var passed = tests.All(test => (string)test.GetType().GetProperty("status")!.GetValue(test)! == "pass");
await File.WriteAllTextAsync("/output/results.json", JsonSerializer.Serialize(new { version = 2, status = passed ? "pass" : "fail", tests }));
EOF

set +e
dotnet restore /tmp/dotnet-rate-limiter-eval/Eval.csproj --ignore-failed-sources >/tmp/dotnet-rate-limiter-restore.log 2>&1
restore_status=$?
if [[ "$restore_status" == "0" ]]; then
  dotnet build /tmp/dotnet-rate-limiter-eval/Eval.csproj --no-restore >/tmp/dotnet-rate-limiter-build.log 2>&1
  build_status=$?
  if [[ "$build_status" == "0" ]]; then
    dotnet /tmp/dotnet-rate-limiter-eval/bin/Debug/net10.0/Eval.dll >/tmp/dotnet-rate-limiter-run.log 2>&1
    run_status=$?
  else
    run_status=$build_status
  fi
else
  run_status=$restore_status
fi
set -e

if [[ "$run_status" != "0" ]]; then
  cat /tmp/dotnet-rate-limiter-restore.log /tmp/dotnet-rate-limiter-build.log /tmp/dotnet-rate-limiter-run.log >&2 || true
  printf '{"version":2,"status":"error","message":"The submitted C# rate limiter did not compile or run."}\n' >/output/results.json
fi

# G9 verification — rate limiter language coverage

Date: 2026-08-25  
Status: **complete**  
Scope: all six executable rate-limiter revisions plus Lab editor coverage

## What changed

`task-family.rate-limiter` now has six runnable revisions that share one
capability, Question Brain bindings, rubric, and learning brief while keeping
language-specific starter files and harnesses:

| Language | Revision | Sandbox profile | Editable file |
| --- | --- | --- | --- |
| Go | `go-rate-limiter-001@1` | Go 1.24 | `main.go` |
| Java | `java-rate-limiter-001@1` | JDK 21 | `RateLimiter.java` |
| TypeScript | `ts-rate-limiter-001@1` | Node.js 24 + strip types | `rate-limiter.ts` |
| JavaScript | `node-rate-limiter-001@1` | Node.js 24 | `rate-limiter.js` |
| C# | `csharp-rate-limiter-001@1` | .NET 10 | `RateLimiter.cs` |
| SQL | `pg-rate-limiter-001@1` | PostgreSQL 17 | `solution.sql` |

The TypeScript and C# revisions are actual runnable tasks, not profile labels:
each has a starter contract, four hidden invariants, a bounded Docker profile,
and a release-pinned immutable hash. They bind to the same `question.q315`
QuestionCard and `capability.distributed-systems.rate-limiter` as the existing
revisions.

## Release pins

- Runtime release: `runtime-task-release-2026-08-25-qb-d00a1493-g9`
- TaskFamily release: `task-family-release-2026-08-25-g9`
- Question Brain release: `question-release-d00a14931e607336`

The workspace launcher and Task Runtime Compose default now select the g9
manifest. Historical g8 remains immutable.

## Runtime smoke

With the runtime Compose stack ready:

```sh
curl -fsS http://127.0.0.1:48227/v1/task-families \
  | jq '.families[] | select(.key == "task-family.rate-limiter") | .revisions'
```

All six revisions were executed through `POST /v1/runs` using the real Docker
sandbox. Each returned HTTP 200 and all four checks passed:

```json
[
  {"taskId":"go-rate-limiter-001","http":200,"status":"pass","tests":["pass","pass","pass","pass"]},
  {"taskId":"ts-rate-limiter-001","http":200,"status":"pass","tests":["pass","pass","pass","pass"]},
  {"taskId":"csharp-rate-limiter-001","http":200,"status":"pass","tests":["pass","pass","pass","pass"]},
  {"taskId":"java-rate-limiter-001","http":200,"status":"pass","tests":["pass","pass","pass","pass"]},
  {"taskId":"node-rate-limiter-001","http":200,"status":"pass","tests":["pass","pass","pass","pass"]},
  {"taskId":"pg-rate-limiter-001","http":200,"status":"pass","tests":["pass","pass","pass","pass"]}
]
```

## Lab editor

The shared CodeMirror language registry now maps `.go` files to the official
`@codemirror/lang-go` grammar. Before this fix `main.go` fell through to
`plain`, so the screenshot showed no token colours even though the editor
theme was loaded. The mapping is covered by the Lab language unit test and the
same editor continues to serve TypeScript, Java, C#, SQL, JavaScript and JSON
grammars.

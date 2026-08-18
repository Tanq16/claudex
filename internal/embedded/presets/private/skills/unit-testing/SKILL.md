---
name: unit-testing
description: Edge-case-driven unit tests for Go and Node. Use when adding or changing tests, when asked to raise coverage, when deciding whether a function needs a test at all, or when reviewing a test file. Triggers on _test.go, *.test.js, go test, node --test, testing.T, node:test, table-driven tests, t.Context(), b.Loop(), and on any request to "add tests".
user-invocable: false
---

# Unit Testing

**Tests pin down the inputs that break things. Coverage is a side effect, never the goal.**

The philosophy is identical in Go and Node; only the runner and the assertion syntax differ.

## Rules

A test earns its place by encoding a scenario you actually reasoned about. Before writing one, work out how the code breaks: boundary values, empty and nil or undefined inputs, zero-length and single-element collections, malformed data, concurrent access, size overflow, and anything that can panic or throw. Those scenarios become the cases.

A test derived from the implementation passes by construction and locks any bug in place, so write both the code and its tests against the intended behavior instead of against each other.

Judge a test by whether it can fail for a real reason. Coverage counts lines executed, which is a different claim from behavior pinned down, and a suite of happy-path assertions reports the higher number while catching less.

A function with no decision logic of its own gets no test. A thin wrapper over a directory creation call or plain arithmetic exercises the language and the standard library rather than your code, and a failure there is not a failure you can act on.

Unit tests exercise one package or module in isolation. Standing up a server, opening a socket, or crossing package boundaries end to end belongs to a separate script, because a unit suite that needs a listening port fails for reasons that have nothing to do with the logic under test.

Mocks are verified once against public documentation or a real response, then committed and pinned as a fixture. A test that reaches a live service at run time is slow and flaky, and one written from memory asserts a shape the service may never have had.

Finish the implementation before adding tests. Tests written against a half-built function describe the scaffolding rather than the behavior.

Table-driven tests are the default shape, because adding the eighth edge case should cost one line rather than one function.

## Go

Tests live in the package they cover, in the same directory, as `package foo` or `package foo_test`.

One `_test.go` file per package is enough. Collect that package's edge cases into a single file rather than mirroring each source file, and split only when one file becomes genuinely unwieldy.

`t.Context()` supplies a test's context, rather than `context.WithCancel(context.Background())`, since it is cancelled when the test ends without a `defer` of your own.

`for b.Loop()` drives the main benchmark loop rather than `for i := 0; i < b.N; i++`, because it excludes setup from the timed region automatically.

`omitzero` rather than `omitempty` in JSON struct tags for `time.Time`, `time.Duration`, structs, slices, and maps. `omitempty` never omits a zero `time.Time`, so the field ships as a meaningless timestamp.

```go
func TestParse(t *testing.T) {
    tests := []struct {
        name    string
        in      string
        want    Result
        wantErr bool
    }{
        {"empty input", "", Result{}, true},
        {"only separator", ",", Result{}, true},
        {"trailing separator", "a,", Result{Items: []string{"a"}}, false},
        {"single element", "a", Result{Items: []string{"a"}}, false},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Parse(t.Context(), tt.in)
            if (err != nil) != tt.wantErr {
                t.Fatalf("Parse(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("Parse(%q) = %v, want %v", tt.in, got, tt.want)
            }
        })
    }
}
```

## Node

Tests live under `test/` as `*.test.js`, or colocated as `*.test.js` beside the module. They run with `node --test`, and `node --test --watch` during development.

The built-in runner covers this, so no test-framework dependency is added. `node:assert/strict` is the assertion import, since the loose variants treat `1` and `'1'` as equal and hide exactly the coercion bugs a test should catch.

A live end-to-end script, such as `test/e2e.mjs` booting the server over HTTP and WebSocket, stays separate from the unit suite and stays optional. Running it as part of `node --test` makes every unit run depend on a free port.

```js
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { parse } from '../src/parse.js';

test('parse', async (t) => {
  const cases = [
    { name: 'empty input', in: '', want: [], throws: true },
    { name: 'only separator', in: ',', want: [], throws: true },
    { name: 'trailing separator', in: 'a,', want: ['a'], throws: false },
    { name: 'single element', in: 'a', want: ['a'], throws: false },
  ];
  for (const c of cases) {
    await t.test(c.name, () => {
      if (c.throws) {
        assert.throws(() => parse(c.in));
        return;
      }
      assert.deepEqual(parse(c.in), c.want);
    });
  }
});
```

Async setup and teardown hang off `before` and `after` inside the test:

```js
import { test, before, after } from 'node:test';
import assert from 'node:assert/strict';

test('handler', async (t) => {
  let store;
  before(() => { store = new Map(); });
  after(() => { store.clear(); });

  await t.test('rejects unknown id', () => {
    assert.throws(() => lookup(store, 'nope'), /not found/);
  });
});
```

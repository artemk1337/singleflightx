# singleflightx

singleflightx is a small, generic Go package that ensures a function is executed only once for a given key, even when called concurrently by multiple goroutines. All concurrent callers receive the same result.

The package is inspired by golang.org/x/sync/singleflight, but focuses on simplicity, generics, and low allocation overhead.

---

## Features

- Deduplicates concurrent executions per key
- Safe for high concurrency
- Generic and type-safe (K comparable, V any)
- Reports whether the result was shared
- Reuses internal objects via sync.Pool

---

## Installation

```bash
go get github.com/yourname/singleflightx
```

---

## Usage

```go
var g singleflightx.Group[string, int]

value, err, shared := g.Do("key", func() (int, error) {
    // expensive operation
    return 42, nil
})

if shared {
    // result was reused from another goroutine
}
```

---

## API

### func (*Group[K, V]) Do(key K, fn func() (V, error)) (V, error, bool)

Executes fn only once for the given key.

If multiple goroutines call Do with the same key concurrently:

- Only one goroutine executes fn
- Other goroutines wait for the result
- All callers receive the same value and error
- The shared return value is true if the result was produced by another goroutine

---

## How It Works

- A mutex protects the map of in-flight calls
- Each call uses a sync.WaitGroup for coordination
- A reference counter tracks waiting goroutines
- Completed calls are cleaned up and reused via sync.Pool

---

## Use Cases

- Cache miss deduplication
- Preventing duplicate database or API requests
- Collapsing concurrent expensive computations

---

## License

MIT

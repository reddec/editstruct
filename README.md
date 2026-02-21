# editstruct

A Go code generation tool for modifying struct field types in-place.

## Use Case

After running a code generator (like `stringer`, `moq`, `sqlc`, or protobuf), you often need to adjust the generated types. `editstruct` lets you declaratively specify type changes in a config file and applies them during `go generate`.

## Installation

```bash
go get -tool github.com/reddec/editstruct
```

## Usage

Create an `edit.yaml` in your package directory:

```yaml
type: Example
fields:
  Total: uint64
---
type: Order
fields:
  CreatedAt: time.Time
  Count: "*int64"
```

Add to your Go file:

```go
//go:generate go tool github.com/reddec/editstruct

type Example struct {
    ID    int64
    Total *int64  // will become uint64
}
```

Run:

```bash
go generate ./...
```

Result:

```go
type Example struct {
    ID    int64
    Total uint64  // changed from *int64 to uint64
}
```

## Config Format

Multi-document YAML where each document specifies one struct:

| Field | Description |
|-------|-------------|
| `type` | Struct name to modify |
| `fields` | Map of field name → new type |

### Type Syntax

- Built-in: `uint64`, `string`, `int`, etc.
- Qualified: `time.Time`, `uuid.UUID` (imports added automatically)
- Pointer: `"*string"` (quote to handle `*` in YAML)
- Slice: `[]int`, `[]string`
- Map: `map[string]int`

## Behavior

- Modifies files in-place
- Preserves comments and struct tags
- Scans only `*.go` files in current directory (non-recursive, excludes `*_test.go`)
- Silently ignores missing fields/structs
- Exits with error on parse failures

> Note: mostly vibe-coded (GLM-5, opencode) but it works

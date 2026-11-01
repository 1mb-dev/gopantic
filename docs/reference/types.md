# Types

Anything `encoding/json` decodes, gopantic decodes the same way. These are the cases worth knowing.

| Type | Behaviour |
|------|-----------|
| Numbers, strings, booleans | Decoded, then [coerced](../guide/coercion.md) if the decoder refused |
| `time.Time` | RFC 3339, plus the other formats in the coercion guide and Unix seconds |
| Structs, pointers | Rules inside them run. A nil pointer has nothing to check |
| Slices, arrays | Rules run on each element |
| `map[K]V` | Rules run on each value. `K` can be a string or an integer, and `interface{}` for YAML, which keeps the key's type |
| `interface{}` | Holds what the decoder produced. `required` works, and nothing goes inside it |
| `json.RawMessage` | Keeps its bytes as they arrived |
| Types with `UnmarshalJSON`, `UnmarshalYAML` or `UnmarshalText` | Decode themselves. `validate` tags still apply to them |

## Absent, null and zero

A plain field can't tell an absent key from a zero value. A pointer can:

```go
type Update struct {
    Name *string `json:"name"` // nil: not sent. "": sent empty
    Age  *int    `json:"age"`
}
```

An absent or `null` slice is nil, and `[]` is empty and not nil, as with `encoding/json`.

## Fixed arrays

Arrays follow `encoding/json`: extra elements are dropped and missing ones stay zero, with no error. To require a length, use a slice and a rule:

```go
Scores []float64 `json:"scores" validate:"len=3"`
```

## Struct tags

`json` tags work as in `encoding/json`, including `json:"-"` and `omitempty`, embedded structs, and case-insensitive key matching. A key that matches no field is ignored.

For YAML, a field's key comes from its `yaml` tag, then its `json` tag, then its lowercased name. `yaml:",inline"` and merge keys behave as in `yaml.v3`. A field with `json:"-"` and no `yaml` tag is skipped on YAML too.

```go
type Config struct {
    Host string `json:"host"`            // key "host" in both
    Port int    `yaml:"port" json:"p"`   // "port" in YAML, "p" in JSON
    TLS  bool                             // key "tls" in YAML, any case of "TLS" in JSON
}
```

## Trees and cycles

A type can refer to itself, and input is a tree, so nothing loops. `Validate` on a value you built with a real cycle stops at the depth limit with an error. Tag a back-pointer `json:"-"` and `Validate` won't follow it:

```go
type Node struct {
    Name     string  `json:"name" validate:"required"`
    Children []*Node `json:"children"`
    Parent   *Node   `json:"-"`
}
```

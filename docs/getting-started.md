# Getting started

```bash
go get github.com/1mb-dev/gopantic
```

Everything lives in one package, `github.com/1mb-dev/gopantic/pkg/model`.

## Parse

Describe the data as a struct. `json` tags name the keys, and `validate` tags hold the rules.

```go
type User struct {
    ID       int    `json:"id" validate:"required,min=1"`
    Username string `json:"username" validate:"required,min=3,max=50"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"min=0,max=150"`
}

data := []byte(`{"id": "123", "username": "alice", "email": "alice@example.com", "age": 25}`)

user, err := model.ParseInto[User](data)
```

`ParseInto` looks at the input to tell JSON from YAML, decodes it, converts values that need it, and validates the result. Here `"123"` became the int `123`. The [coercion guide](guide/coercion.md) lists what converts.

## YAML

The same struct and the same call work on YAML, and `json` tags apply to it too:

```go
user, err := model.ParseInto[User]([]byte(`
id: 456
username: bob
email: bob@example.com
age: 30
`))
```

Detection reads the first byte, so `{` or `[` always means JSON. If you know the format, say so:

```go
user, err := model.ParseIntoWithFormat[User](data, model.FormatYAML)
```

## Errors

A failed parse returns the zero value and an error covering every problem found, up to 100.

```go
_, err := model.ParseInto[User]([]byte(`{"id": "x", "username": "al", "email": "alice@example.com"}`))
fmt.Println(err)
// multiple errors: parse error on field "ID": cannot parse string "x" as integer: ...;
// validation error on field "Username": string length must be at least 3 characters
```

To act on each problem, use `errors.As` for the first of a kind, or walk the list:

```go
var list model.ErrorList
if errors.As(err, &list) {
    for _, v := range list.ValidationErrors() {
        fmt.Println(v.FieldPath, v.Rule) // Username min
    }
}
```

Error messages can quote the input. Values of [sensitive fields](guide/validation.md#sensitive-values) are kept out, but don't send messages to clients.

## Validate without parsing

`Validate` runs the same rules on a struct you filled in yourself:

```go
user := User{ID: 7, Username: "alice", Email: "alice@example.com"}
err := model.Validate(&user)
```

## Limits

Inputs over 10 MB, and input nested past 64 levels, are rejected before decoding. Rules stop at 32 levels of nested structs, with an error. All three can be changed:

```go
model.SetMaxInputSize(5 * 1024 * 1024)
model.SetMaxStructureDepth(32)
model.SetMaxValidationDepth(16)
```

## Next

The [validation guide](guide/validation.md) covers every rule and how to add your own. To parse the same bytes repeatedly, see [caching](guide/caching.md).

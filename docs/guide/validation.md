# Validation

Rules go in a `validate` tag, separated by commas. All of them run, and all failures are reported.

```go
type User struct {
    Name  string `json:"name" validate:"required,min=2,max=50"`
    Email string `json:"email" validate:"required,email"`
}
```

## Built-in rules

| Rule | Meaning |
|------|---------|
| `required` | Not the zero value |
| `min=N`, `max=N` | Bounds on a number, or on the length of a string, slice or array |
| `length=N`, or `len=N` | Exact length of a string, slice or array |
| `email` | Looks like an email address |
| `alpha` | Letters a to z only |
| `alphanum` | Letters and digits only |

String lengths count bytes. An empty string fails `required` and passes `min=0`, and `required` can't tell an absent `0` from a sent one. Use a pointer when that matters:

```go
Age *int `json:"age" validate:"required"` // nil fails, 0 passes
```

## Where rules apply

Rules run wherever the type holds them: nested structs, pointers, the elements of slices and arrays, map values, and top-level slices and maps. So `model.ParseInto[[]User](data)` checks every user.

```go
type Order struct {
    Items []Item          `json:"items"` // each Item is validated
    Meta  map[string]Item `json:"meta"`  // each value is validated
}
```

Errors from nested values carry the full path in `FieldPath`, such as `Items[1].Name`, and the leaf name in `Field`.

## A tag that cannot run is an error

A misspelled rule would otherwise leave a field unchecked with no sign of it. So an unknown rule, a missing or malformed parameter, or a stray comma fails every parse and every `Validate` for that type, even when the input never reaches the field:

```
main.Order.Name: invalid validate tag "requird": unknown validator "requird":
use one of alpha, alphanum, email, len, length, max, min, required, or register it first
```

This also catches rules copied from go-playground/validator that gopantic doesn't have. Register those yourself, as below.

## Your own rules

Register a function under a name, then use the name in tags. Do this before parsing, or at any point: registering clears the cached rules, so types parsed earlier pick it up.

```go
model.RegisterGlobalFunc("is_even", func(field string, value interface{}, params map[string]interface{}) error {
    if n, ok := value.(int); ok && n%2 != 0 {
        return model.NewValidationError(field, value, "is_even", "must be even")
    }
    return nil
})

type Numbers struct {
    Count int `json:"count" validate:"is_even"`
}
```

The text after `=` arrives as `params["value"]`, a `float64` if it reads as a number and a string otherwise:

```go
model.RegisterGlobalFunc("oneof", func(field string, value interface{}, params map[string]interface{}) error {
    allowed, _ := params["value"].(string)
    for _, option := range strings.Fields(allowed) {
        if value == option {
            return nil
        }
    }
    return model.NewValidationError(field, value, "oneof", "must be one of: "+allowed)
})

type Post struct {
    Status string `json:"status" validate:"oneof=draft published archived"`
}
```

To compare fields, register a cross-field rule, which also receives the struct:

```go
model.RegisterGlobalCrossFieldFunc("eqfield", func(field string, value interface{}, parent reflect.Value, params map[string]interface{}) error {
    other := parent.FieldByName(params["value"].(string))
    if !other.IsValid() || other.Interface() != value {
        return model.NewValidationError(field, value, "eqfield", "fields do not match")
    }
    return nil
})

type Registration struct {
    Password string `json:"password" validate:"required,min=8"`
    Confirm  string `json:"confirm" validate:"required,eqfield=Password"`
}
```

## Fields input cannot set

A field tagged `json:"-"` is often filled in by your code, from a URL parameter or a session. A parse can't set it, so a parse doesn't check it. Call `Validate` once you have:

```go
type CreateChat struct {
    Message string `json:"message" validate:"required"`
    KBID    string `json:"-" validate:"required"`
}

req, err := model.ParseIntoWithFormat[CreateChat](body, model.FormatJSON) // KBID is not checked here
if err != nil {
    return err
}
req.KBID = kbIDFromPath(r)
err = model.Validate(&req) // now it is
```

The rule is the same for any field: a parse skips rules on fields neither decoder can set, and `Validate` runs them. Whether a field can be set doesn't depend on which format arrived, so a field with `json:"-" yaml:"x"` keeps its rules on JSON input too, and `required` there fails every JSON parse.

`Validate` runs a field's own rules but doesn't go inside such a field. A `Parent *Node` tagged `json:"-"` is usually a way back up a tree, and it may point at a type from another library. To check what is inside, call `Validate` on that value.

## Errors

A failed parse returns a `model.ErrorList`, which works with `errors.As`:

```go
var v *model.ValidationError
if errors.As(err, &v) {
    fmt.Println(v.FieldPath, v.Rule, v.Message)
}
```

`errors.As` stops at the first match. `list.ValidationErrors()` returns them all, `list.GroupByField()` groups them, and `list.ToJSON()` renders the validation errors as a report for an API. Conversion failures are `*model.ParseError`, and aren't in that report, so check for them before returning one. A parse reports at most 100 errors and then one saying it stopped.

## Sensitive values

Errors keep the rejected value in their `Value` field, or `"object"` or `"array"` in place of a rejected container, since a secret could sit inside one. `SanitizedValue()` returns `[REDACTED]` instead when the field is sensitive or the value is a slice, map or struct, and `ToJSON` uses it. A `ParseError` on a sensitive field also leaves the value out of its message, which would otherwise quote it.

A field is sensitive when it, or anything above it, has a Go name or input key containing `password`, `passwd`, `secret`, `token`, `key`, `credential`, `auth`, `api_key`, `apikey`, `private` or `bearer`, or is tagged `json:"-"`. The match ignores case and looks anywhere in the name, so `Author` and `Monkey` count too. Change the list with `model.SetSensitiveFieldPatterns` or `model.AddSensitiveFieldPattern`.

Log `SanitizedValue()`, never `Value`.

# Coming from encoding/json and validator

Upgrading from an older gopantic instead? The [v1.5.0 changelog](changelog.md) lists what changed.

The usual code:

```go
var user User
if err := json.Unmarshal(data, &user); err != nil {
    return User{}, err
}
if err := validate.Struct(user); err != nil {
    return User{}, err
}
return user, nil
```

becomes:

```go
return model.ParseInto[User](data)
```

Your structs keep their `json` tags. Three things need attention: the `validate` tags, fields your code fills in, and error handling.

## Tags

gopantic has fewer rules than validator, and an unknown one is an error on every parse, so you will find out at once.

| validator | gopantic |
|-----------|----------|
| `required`, `min`, `max`, `len`, `email`, `alpha`, `alphanum` | The same names, with the differences below |
| `dive` | Remove it. Rules on elements and map values always run |
| `omitempty` | Remove it, and [register](guide/validation.md#your-own-rules) a rule that accepts empty input if you need one |
| `eqfield`, `gtfield`, `nefield`, `oneof`, `url`, `uuid`, others | Not built in. Register your own |

`required` differs for pointers: validator and gopantic both fail a nil pointer, and gopantic passes a pointer to a zero value. Validator's `email` is stricter in places, and gopantic's is a format check, not a full RFC parser.

## Fields your code fills in

With validator, this runs on whatever the struct holds when you call it:

```go
KBID string `json:"-" validate:"required"`
```

`ParseInto` validates during the parse, before your code has set `KBID`, so it skips that rule. Keep the tag and call `model.Validate` after filling the field in. [Fields input cannot set](guide/validation.md#fields-input-cannot-set) has the details, and this is the one place where dropping the second call would silently lose a check.

## Coercion

gopantic accepts `{"age": "25"}` for an `int`, where `encoding/json` errors. If your API should reject that, keep `json.Unmarshal` and call `model.Validate` on its result. You keep the same rules and the strict decoder.

## Errors

```go
// validator
var errs validator.ValidationErrors
if errors.As(err, &errs) {
    for _, e := range errs {
        fmt.Println(e.Field(), e.Tag())
    }
}

// gopantic
var list model.ErrorList
if errors.As(err, &list) {
    for _, e := range list.ValidationErrors() {
        fmt.Println(e.FieldPath, e.Rule)
    }
}
```

A list can also hold `*model.ParseError` for values that could not be converted.

## Custom rules

```go
// validator
validate.RegisterValidation("strong", func(fl validator.FieldLevel) bool {
    return isStrong(fl.Field().String())
})

// gopantic
model.RegisterGlobalFunc("strong", func(field string, value interface{}, params map[string]interface{}) error {
    if s, ok := value.(string); ok && !isStrong(s) {
        return model.NewValidationError(field, value, "strong", "password is too weak")
    }
    return nil
})
```

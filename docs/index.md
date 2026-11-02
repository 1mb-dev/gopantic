# gopantic

Parse JSON or YAML into a Go struct, convert what can be converted exactly, and run the rules in its `validate` tags, in one call. You get back the value the input meant, or an error.

```go
type User struct {
    ID    int    `json:"id" validate:"required,min=1"`
    Name  string `json:"name" validate:"required,min=2"`
    Email string `json:"email" validate:"required,email"`
}

user, err := model.ParseInto[User]([]byte(`{"id": "42", "name": "Alice", "email": "alice@example.com"}`))
// user.ID == 42
```

The string `"42"` becomes the int `42` because that conversion loses nothing. `"42.5"` wouldn't fit an int, so it's an error. The same struct reads YAML, and [naming the format](getting-started.md#yaml) avoids detection missing a short document.

## Who it is for

gopantic suits code that reads input it doesn't control: API payloads where numbers arrive as strings, config files that may be JSON or YAML, data imports. You write one struct and get decoding, conversion and validation together, with every problem reported at once.

If your input is already well typed and JSON only, use `encoding/json` with [go-playground/validator](https://github.com/go-playground/validator). It has far more rules, and you wouldn't be using what gopantic adds. The two are within 30% of each other on correctly typed JSON, so speed won't decide it. Input that needs converting takes about 3 times as long; see [Benchmarks](benchmarks.md).

## Install

```bash
go get github.com/1mb-dev/gopantic
```

Requires Go 1.24 or later. The one dependency is `gopkg.in/yaml.v3`.

## Where to go next

Start with [Getting started](getting-started.md). The [Validation](guide/validation.md) and [Coercion](guide/coercion.md) guides are the two pages you will return to. [Known limitations](known-limitations.md) lists what gopantic doesn't do, and the [changelog](changelog.md) lists what v1.5.0 changed.

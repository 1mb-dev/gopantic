# gopantic

[![Go Reference](https://pkg.go.dev/badge/github.com/1mb-dev/gopantic.svg)](https://pkg.go.dev/github.com/1mb-dev/gopantic)
[![CI](https://github.com/1mb-dev/gopantic/actions/workflows/ci.yml/badge.svg)](https://github.com/1mb-dev/gopantic/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

Parse JSON or YAML into a Go struct, convert what can be converted exactly, and run the rules in its `validate` tags, in one call. You get back the value the input meant, or an error.

```go
type User struct {
    ID    int    `json:"id" validate:"required,min=1"`
    Name  string `json:"name" validate:"required,min=2"`
    Email string `json:"email" validate:"required,email"`
    Age   int    `json:"age" validate:"min=18,max=120"`
}

raw := []byte(`{"id": "42", "name": "Alice", "email": "alice@example.com", "age": "28"}`)

user, err := model.ParseInto[User](raw)
// user.ID == 42, user.Age == 28
```

The same struct takes YAML with no other change. Input that doesn't fit is an error:

```go
_, err := model.ParseInto[User]([]byte(`{"id": 1.5, "name": "A", "email": "nope", "age": 28}`))
// multiple errors: parse error on field "ID": value has a fractional part;
// validation error on field "Name": string length must be at least 2 characters;
// validation error on field "Email": invalid email address format
```

## Who it is for

gopantic suits code that reads input it doesn't control: API payloads where numbers arrive as strings, config files that may be JSON or YAML, data imports. You write one struct and get decoding, conversion and validation together, with every problem reported at once.

If your input is already well typed and JSON only, use `encoding/json` with [go-playground/validator](https://github.com/go-playground/validator). It has far more rules, and you wouldn't be using what gopantic adds. The two are within 25% of each other on correctly typed JSON, so speed won't decide it. Input that needs converting takes about 3.5 times as long. Figures are on the [benchmarks page](https://1mb-dev.github.io/gopantic/benchmarks/).

## Install

```bash
go get github.com/1mb-dev/gopantic
```

Requires Go 1.24 or later. The one dependency is `gopkg.in/yaml.v3`.

## Documentation

The [full documentation](https://1mb-dev.github.io/gopantic/) has the validation rules, what converts and what errors, caching, and the API. Upgrading from v1.4 can reject input that used to pass, so read the [v1.5.0 changelog](https://1mb-dev.github.io/gopantic/changelog/) first. [Known limitations](https://1mb-dev.github.io/gopantic/known-limitations/) lists what gopantic doesn't do. Runnable programs are in [examples/](examples/).

## Contributing

See [CONTRIBUTING](.github/CONTRIBUTING.md). Report security issues as described in the [security policy](.github/SECURITY.md).

## License

MIT

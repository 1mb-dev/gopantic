# Coercion

gopantic hands input to the standard decoder first. Only when that decoder rejects a value does coercion look at it, and it converts the value only if nothing is lost. Input that needs no conversion is decoded exactly as `encoding/json` or `yaml.v3` would decode it, and one field needing conversion doesn't change how the others decode.

## What converts

| Into | From | Example |
|------|------|---------|
| Integer kinds | Number strings, whole floats | `"42"` and `42.0` to `42` |
| Floats | Number strings, integers | `"3.14"` to `3.14` |
| `bool` | The strings below, numbers | `"yes"` to `true`, `0` to `false` |
| `string` | Numbers, booleans | `42` to `"42"`, `true` to `"true"` |
| `time.Time` | Date strings, Unix seconds | `1704067200` |

The strings accepted as booleans are `true`, `1`, `yes`, `on` and `false`, `0`, `no`, `off`, in lower, title or upper case. The empty string is false.

Times parse as RFC 3339, `2006-01-02T15:04:05`, `2006-01-02 15:04:05` or `2006-01-02`. Unix seconds have to fall in years 1 to 9999.

The rules are the same inside slices, arrays, pointers and map values, and in map keys, so `map[int]string` reads `{"1": "a"}`.

## What errors

Coercion refuses anything that would change the value:

| Input | Target | Why |
|-------|--------|-----|
| `300` | `int8` | Out of range |
| `4.7` | `int` | Fractional |
| `-1` | `uint` | Negative |
| `1e60` | `float32` | Out of range |
| `1e16` | `int` | Written as a float at or past 2^53, so it may have been rounded already |
| `"NaN"`, `"Inf"` | Any float | Not a number gopantic will return |
| `{"a": 1}` | `string` | Would hand back Go syntax as if it were the input |
| `1e300` | `time.Time` | Past year 9999 |
| `abc` | `json.Number` | Not a number |

Whole numbers written out in full are read digit by digit, so `9007199254740993` stays exact in an `int64`. The rules are the same in YAML, where `yaml.v3` alone would turn `1.9` into `1`. A `json.Number` keeps the digits as written.

## Types that decode themselves

Fields with an `UnmarshalJSON`, `UnmarshalYAML` or `UnmarshalText` method decode themselves, and `json.RawMessage` keeps its bytes as they arrived. Coercion leaves them alone, with one exception: `time.Time` falls back to the wider list of formats above.

## Coercion cannot be turned off

If a string in a numeric field should be an error, gopantic is the wrong tool for that field. Decode with `encoding/json`, then call `model.Validate`:

```go
var req Request
if err := json.Unmarshal(body, &req); err != nil {
    return err
}
err := model.Validate(&req)
```

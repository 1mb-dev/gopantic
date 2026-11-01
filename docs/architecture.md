# How it works

All the code is one package, `pkg/model`:

| File | Holds |
|------|-------|
| `parse.go` | `ParseInto`, `Validate`, the validation walk |
| `decode.go` | The type-directed walker and field tables |
| `coerce.go` | Scalar conversion |
| `validate.go` | Tags, the registry, per-type rule analysis |
| `validators.go` | Built-in rules |
| `format.go` | Format detection, depth checks |
| `cache.go` | `CachedParser` |
| `errors.go`, `config.go` | Error types, limits |

## Decoding

JSON tries the fast path first. A byte scan checks the depth limit, then `json.Unmarshal` decodes straight into `T`. If that works, nothing else in the decoder runs, so correctly typed input costs little more than `encoding/json`.

When the decoder rejects a value, the input is decoded again by the walker. It builds a tree of raw byte ranges, then walks the target type, not the input. Each leaf goes to the standard decoder first, and coercion only sees what that decoder refused. So a `json.RawMessage` keeps its bytes and an `UnmarshalJSON` method runs, whether or not some other field needed coercion.

YAML always takes the walker, because `yaml.v3` ignores `json` tags and decoding straight into `T` would drop those fields. `yaml.v3` enforces its alias limits only on a whole-document decode, so one runs first and its result goes unused, except for the depth check.

A differential test compares each parse with `encoding/json` or `yaml.v3`, with and without a field that forces coercion. Fuzzing does the same on generated input.

## Validating

Every parse and every `Validate` goes through one walk, in `parse.go`. There is only one so that the rules that run can't depend on how a value was decoded.

Rules are parsed once per type and cached. That pass looks only at types, and records two things across everything reachable: tags that can't run, and whether any rule exists. So a typo fails under an empty slice, and a type with no rules skips the walk.

Paths for errors are built only when an error needs one. Building them as the walk descends would copy every enclosing map key once per element, and one long key above a large slice would turn that quadratic.

## Choices

`ParseInto[User]` returns a `User`, with no build step and no `interface{}` to assert. Code generators are faster, and that's the cost of having none.

Rules live on the struct, so they can't drift from it. A tag is a short string, though, which is why anything complicated becomes a registered function.

Coercion exists to accept input that means something obvious, and it stops where the meaning stops. An `int8` can't hold 300, and returning 44 would be worse than failing. A misspelled tag is an error for the same reason: a rule that silently does nothing is a bug you can't see.

A parse reports every error it finds, so a client can fix them in one round. The cap of 100 is there because one error per element would otherwise let a large input allocate without limit.

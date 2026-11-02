# Known limitations

Things gopantic doesn't do, or does differently from what you might assume. Most are deliberate. A few are costs accepted for now.

## Dependencies

`gopkg.in/yaml.v3` is archived upstream and gets no more changes. It works, but a bug in it would mean moving to a maintained fork.

## Coercion

You can't turn it off. A string in a numeric field is converted if it can be, and [the coercion guide](guide/coercion.md#coercion-cannot-be-turned-off) shows what to do instead.

Floats at or past 2^53 are rejected for integer fields even when they look whole: `1e16` and `9007199254740992.0` both error, since a float that large may already have been rounded. Writing the digits out works, so send `10000000000000000`.

## Validation

If you parse into a type with a rule on a `json:"-"` field and never call `Validate`, that rule doesn't run. See [Fields input cannot set](guide/validation.md#fields-input-cannot-set). `Validate` runs such a field's own rules but doesn't go inside it, so call `Validate` on the inner value if it needs checking.

`required` means not zero, so it can't tell a sent `0` from an absent key. Use a pointer.

Rules inside whatever an `interface{}` field holds don't run.

The built-in set is small. There's no `url`, `uuid`, `oneof` or `gt`, and unknown rules are errors, so anything else has to be [registered](guide/validation.md#your-own-rules).

A value with a real pointer loop, not tagged `json:"-"`, stops at the depth limit with an error, though nothing is wrong with it.

An element that fails to decode can be reported twice: `{"items":[5]}` into a `[]Item` gives the conversion error, and also a `required` error for the zero value. The same goes for a map key with more than one spelling, like `"007"` into a `map[int]Item`. No bad value gets through, but you may see an extra line.

## Formats

An inline map needs string keys: `map[interface{}]T` with `yaml:",inline"` errors on every key, as it does in `yaml.v3`.

Input starting with `{` or `[` is treated as JSON, so one-line YAML like `{a: 1}` is a syntax error under `ParseInto`. Use `ParseIntoWithFormat` with `FormatYAML`.

A YAML document can't need more than 50,000,000 key comparisons, because `yaml.v3` compares every pair of keys in a mapping, and again for each alias of it. That's one mapping of 10,000 keys, or 100 of 1,000. `SetMaxYAMLKeyComparisons` changes it.

YAML always takes the slow path through the walker, because `yaml.v3` ignores `json` tags. [Benchmarks](benchmarks.md) has the cost.

## Sensitive values

Matching is by substring on names. It over-redacts `Author`, and it can't know `Hash` is secret unless the field is tagged `json:"-"` or you add a pattern.

Long map keys match on their first 64 bytes, because scanning whole keys would let one long key cost time on every error. A key that only looks sensitive past that point isn't redacted.

Errors keep the rejected value in `Value`. Log `SanitizedValue()`.

## Caching

`MaxEntries` counts entries. There's no memory limit, and eviction scans the whole cache, which is fine at the default of 1000 and slower far past it.

A hit copies through reflection, so the saving shrinks as a type holds more slices and maps. A large `map[string]any` can take almost as long to copy as to parse, and allocate as much.

Only `time.Time` is treated as immutable. Other types holding hidden pointers aren't cached, even when sharing them would be safe.

A hit skips validation; see [Keys and staleness](guide/caching.md#keys-and-staleness).

package tests

import (
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/1mb-dev/gopantic/pkg/model"
)

type ctCacheLeaf struct {
	Vals []int `json:"vals"`
}

// CtCachePromoted is exported so encoding/json promotes its fields through embedding.
type CtCachePromoted struct {
	Promoted []string `json:"promoted"`
}

type ctCacheEmbeddedHidden struct {
	Hidden []string `json:"hidden"`
}

type ctCacheAll struct {
	CtCachePromoted
	ctCacheEmbeddedHidden
	Slice  []string          `json:"slice"`
	Ptr    *ctCacheLeaf      `json:"ptr"`
	Map    map[string]*int   `json:"map"`
	Any    any               `json:"any"`
	Raw    json.RawMessage   `json:"raw"`
	Array  [1]ctCacheLeaf    `json:"array"`
	Nested []ctCacheLeaf     `json:"nested"`
	At     time.Time         `json:"at"`
	Next   *ctCacheAll       `json:"next"`
	Tree   map[string]any    `json:"tree"`
	Empty  []string          `json:"empty"`
	Nil    map[string]string `json:"nil"`
}

type ctCacheValues struct {
	N    int       `json:"n"`
	S    string    `json:"s"`
	At   time.Time `json:"at"`
	Pair [2]int    `json:"pair"`
}

// ctCacheHidden keeps a reference the cache cannot copy.
type ctCacheHidden struct {
	data []string
}

func (h *ctCacheHidden) UnmarshalJSON(raw []byte) error {
	h.data = []string{string(raw)}
	return nil
}

type ctCacheBig struct {
	N *big.Int `json:"n"`
}

// ctCacheTwice reaches the same type twice, and only the second field is out of reach.
type ctCacheTwice struct {
	Shown  ctCacheLeaf `json:"shown"`
	hidden ctCacheLeaf
}

func (c *ctCacheTwice) UnmarshalJSON(raw []byte) error {
	c.hidden.Vals = []int{len(raw)}
	return nil
}

const ctCacheInput = `{
	"promoted":["p"],"hidden":["h"],"slice":["a"],"ptr":{"vals":[1]},"map":{"k":1},
	"any":{"list":[{"x":1}]},"raw":{"a":1},"array":[{"vals":[2]}],"nested":[{"vals":[3]}],
	"at":"2026-01-02T03:04:05Z","next":{"slice":["deep"]},"tree":{"t":["u"]},"empty":[]
}`

func scrambleCacheAll(v *ctCacheAll) {
	v.Promoted[0] = "mutated"
	v.Hidden[0] = "mutated"
	v.Slice[0] = "mutated"
	v.Ptr.Vals[0] = 99
	*v.Map["k"] = 99
	v.Any.(map[string]any)["list"].([]any)[0].(map[string]any)["x"] = 99
	v.Raw[1] = 'Z'
	v.Array[0].Vals[0] = 99
	v.Nested[0].Vals[0] = 99
	v.Next.Slice[0] = "mutated"
	v.Tree["t"].([]any)[0] = "mutated"
}

func TestCachedParserResultsShareNothing(t *testing.T) {
	want, err := model.ParseInto[ctCacheAll]([]byte(ctCacheInput))
	if err != nil {
		t.Fatal(err)
	}

	cached := model.NewCachedParser[ctCacheAll](nil)
	defer cached.Close()

	for _, step := range []string{"miss", "hit", "second hit"} {
		got, err := cached.Parse([]byte(ctCacheInput))
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s: got %+v, want %+v", step, got, want)
		}
		if got.Empty == nil || got.Nil != nil {
			t.Fatalf("%s: empty and nil must stay distinct: %#v, %#v", step, got.Empty, got.Nil)
		}
		scrambleCacheAll(&got)
	}

	if _, _, rate := cached.Stats(); rate == 0 {
		t.Fatal("no hits: this test would prove nothing")
	}
}

func TestCachedParserInterfaceTarget(t *testing.T) {
	cached := model.NewCachedParser[any](nil)
	defer cached.Close()

	input := []byte(`{"a":["b"]}`)
	first, err := cached.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	first.(map[string]any)["a"].([]any)[0] = "mutated"

	second, err := cached.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	if got := second.(map[string]any)["a"].([]any)[0]; got != "b" {
		t.Fatalf("cached value changed to %v", got)
	}
}

func TestCachedParserValueTypesHit(t *testing.T) {
	cached := model.NewCachedParser[ctCacheValues](nil)
	defer cached.Close()

	input := []byte(`{"n":1,"s":"x","at":"2026-01-02T03:04:05Z","pair":[1,2]}`)
	for i := 0; i < 2; i++ {
		if _, err := cached.Parse(input); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, rate := cached.Stats(); rate != 0.5 {
		t.Fatalf("hit rate %v, want 0.5", rate)
	}
}

func TestCachedParserBypassesUncopyableTypes(t *testing.T) {
	t.Run("unexported slice", func(t *testing.T) {
		cached := model.NewCachedParser[ctCacheHidden](nil)
		defer cached.Close()

		input := []byte(`"a"`)
		first, err := cached.Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		second, err := cached.Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		if &first.data[0] == &second.data[0] {
			t.Fatal("results share a backing array")
		}
		if size, _, rate := cached.Stats(); size != 0 || rate != 0 {
			t.Fatalf("size %d, hit rate %v: type should bypass the cache", size, rate)
		}
	})

	t.Run("unexported field of a type already seen", func(t *testing.T) {
		cached := model.NewCachedParser[ctCacheTwice](nil)
		defer cached.Close()

		input := []byte(`{}`)
		first, err := cached.Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		second, err := cached.Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		if &first.hidden.Vals[0] == &second.hidden.Vals[0] {
			t.Fatal("results share a backing array")
		}
	})

	t.Run("big.Int behind a pointer", func(t *testing.T) {
		cached := model.NewCachedParser[ctCacheBig](nil)
		defer cached.Close()

		input := []byte(`{"n":123}`)
		first, err := cached.Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		first.N.SetInt64(9)
		second, err := cached.Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		if second.N.Int64() != 123 {
			t.Fatalf("got %v", second.N)
		}
	})
}

func TestCachedParserKeysOnFormat(t *testing.T) {
	cached := model.NewCachedParser[map[string]any](nil)
	defer cached.Close()

	// Valid in both formats, with different values.
	input := []byte(`{"a": 0x10}`)
	if _, err := cached.ParseWithFormat(input, model.FormatJSON); err == nil {
		t.Fatal("JSON accepted a hex literal")
	}
	got, err := cached.ParseWithFormat(input, model.FormatYAML)
	if err != nil {
		t.Fatal(err)
	}
	if got["a"] != 16 {
		t.Fatalf("got %v", got["a"])
	}
}

func TestCachedParserChecksSizeFirst(t *testing.T) {
	orig := model.GetMaxInputSize()
	model.SetMaxInputSize(8)
	defer model.SetMaxInputSize(orig)

	cached := model.NewCachedParser[ctCacheValues](nil)
	defer cached.Close()

	input := []byte(`{"n":1,"s":"too long"}`)
	if _, err := cached.Parse(input); err == nil || !strings.Contains(err.Error(), "exceeds maximum") {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := cached.ParseWithFormat(input, model.FormatJSON); err == nil || !strings.Contains(err.Error(), "exceeds maximum") {
		t.Fatalf("ParseWithFormat: %v", err)
	}
}

func TestCachedParserCloseIsConcurrent(t *testing.T) {
	cached := model.NewCachedParser[ctCacheValues](nil)
	var closers sync.WaitGroup
	for i := 0; i < 8; i++ {
		closers.Add(1)
		go func() {
			defer closers.Done()
			cached.Close()
		}()
	}
	closers.Wait()
}

// Each goroutine owns its result, so writing to it must not race with the others.
func TestCachedParserConcurrentMutation(t *testing.T) {
	cached := model.NewCachedParser[ctCacheAll](nil)
	defer cached.Close()

	var parsers sync.WaitGroup
	for i := 0; i < 8; i++ {
		parsers.Add(1)
		go func() {
			defer parsers.Done()
			for j := 0; j < 50; j++ {
				got, err := cached.Parse([]byte(ctCacheInput))
				if err != nil {
					t.Error(err)
					return
				}
				if got.Slice[0] != "a" {
					t.Errorf("got %q", got.Slice[0])
					return
				}
				scrambleCacheAll(&got)
			}
		}()
	}
	parsers.Wait()
}

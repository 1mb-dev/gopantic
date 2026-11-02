package tests

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// allocated reports the bytes run allocates. The guard exists to stop work before it starts, so
// this is the signal: time would make these tests depend on the machine.
func allocated(run func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	run()
	runtime.ReadMemStats(&after)
	return after.TotalAlloc - before.TotalAlloc
}

func flatKeys(keys int) string {
	var b strings.Builder
	for i := 0; i < keys; i++ {
		fmt.Fprintf(&b, "k%d: 1\n", i)
	}
	return b.String()
}

func keyedMaps(maps, keys int) string {
	var b strings.Builder
	for m := 0; m < maps; m++ {
		fmt.Fprintf(&b, "m%d:\n", m)
		for i := 0; i < keys; i++ {
			fmt.Fprintf(&b, "  k%d: 1\n", i)
		}
	}
	return b.String()
}

func aliasedKeys(keys, refs int) string {
	var b strings.Builder
	b.WriteString("base: &b\n")
	for i := 0; i < keys; i++ {
		fmt.Fprintf(&b, "  k%d: 1\n", i)
	}
	b.WriteString("use:\n")
	for i := 0; i < refs; i++ {
		b.WriteString("  - *b\n")
	}
	return b.String()
}

func mergedKeys(keys, refs int) string {
	var b strings.Builder
	b.WriteString("base: &b\n")
	for i := 0; i < keys; i++ {
		fmt.Fprintf(&b, "  k%d: 1\n", i)
	}
	for i := 0; i < refs; i++ {
		fmt.Fprintf(&b, "m%d:\n  <<: *b\n", i)
	}
	return b.String()
}

func TestYAMLGuardRejectsDuplicatesBeforeDecoding(t *testing.T) {
	input := []byte(strings.Repeat("a: 1\n", 10000))

	var err error
	used := allocated(func() {
		_, err = model.ParseIntoWithFormat[map[string]int](input, model.FormatYAML)
	})
	if err == nil || !strings.Contains(err.Error(), "already defined") {
		t.Fatalf("got %.120v", err)
	}
	if used > 32<<20 {
		t.Fatalf("allocated %d MB: the decode ran", used>>20)
	}
}

func TestYAMLGuardCoversTheParserRoute(t *testing.T) {
	input := []byte(strings.Repeat("a: 1\n", 10000))

	var err error
	used := allocated(func() { _, err = model.GetParser(model.FormatYAML).Parse(input) })
	if err == nil || !strings.Contains(err.Error(), "already defined") {
		t.Fatalf("got %.120v", err)
	}
	if used > 32<<20 {
		t.Fatalf("allocated %d MB: the decode ran", used>>20)
	}
}

func TestYAMLGuardRejectsADuplicateAnywhere(t *testing.T) {
	for _, input := range []string{
		"a: 1\nb: 2\na: 3\n",
		"x:\n  y:\n    a: 1\n    a: 2\n",
		"- a: 1\n  a: 2\n",
		"<<: {a: 1}\n<<: {b: 2}\n",
	} {
		_, err := model.ParseIntoWithFormat[interface{}]([]byte(input), model.FormatYAML)
		if err == nil || !strings.Contains(err.Error(), "already defined") {
			t.Errorf("%q: got %v", input, err)
		}
	}
}

func TestYAMLGuardChargesAllComparisons(t *testing.T) {
	for name, input := range map[string]string{
		"one large mapping":   flatKeys(20000),
		"many mappings":       keyedMaps(30, 5000),
		"aliases of a map":    aliasedKeys(5000, 12),
		"merges of a map":     mergedKeys(5000, 12),
		"nested alias groups": "a: &a\n" + "  " + strings.ReplaceAll(strings.TrimRight(flatKeys(5000), "\n"), "\n", "\n  ") + "\nb: &b [*a, *a, *a, *a]\nc: &c [*b, *b, *b, *b]\n",
	} {
		t.Run(name, func(t *testing.T) {
			var err error
			used := allocated(func() {
				_, err = model.ParseIntoWithFormat[interface{}]([]byte(input), model.FormatYAML)
			})
			if err == nil || !strings.Contains(err.Error(), "SetMaxYAMLKeyComparisons") {
				t.Fatalf("got %.120v", err)
			}
			if used > 256<<20 {
				t.Fatalf("allocated %d MB: the decode ran", used>>20)
			}
		})
	}
}

func TestYAMLGuardAllowsOrdinaryDocuments(t *testing.T) {
	for name, input := range map[string]string{
		"one mapping at the budget": flatKeys(10000),
		"many small mappings":       keyedMaps(1000, 100),
		"a few aliases":             aliasedKeys(1000, 20),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := model.ParseIntoWithFormat[interface{}]([]byte(input), model.FormatYAML); err != nil {
				t.Fatalf("got %.120v", err)
			}
		})
	}
}

func TestYAMLGuardEndsOnACycle(t *testing.T) {
	_, err := model.ParseIntoWithFormat[interface{}]([]byte("a: &x [*x]\n"), model.FormatYAML)
	if err == nil {
		t.Fatal("no error")
	}
}

func TestYAMLKeyComparisonsSetting(t *testing.T) {
	orig := model.GetMaxYAMLKeyComparisons()
	defer model.SetMaxYAMLKeyComparisons(orig)

	if orig != 50_000_000 {
		t.Fatalf("default is %d", orig)
	}

	model.SetMaxYAMLKeyComparisons(10)
	if _, err := model.ParseIntoWithFormat[interface{}]([]byte(flatKeys(6)), model.FormatYAML); err == nil {
		t.Fatal("15 comparisons passed a limit of 10")
	}
	if _, err := model.ParseIntoWithFormat[interface{}]([]byte(flatKeys(5)), model.FormatYAML); err != nil {
		t.Fatalf("10 comparisons failed a limit of 10: %v", err)
	}

	model.SetMaxYAMLKeyComparisons(0)
	if _, err := model.ParseIntoWithFormat[interface{}]([]byte(flatKeys(11000)), model.FormatYAML); err != nil {
		t.Fatalf("budget off: %.120v", err)
	}
	if _, err := model.ParseIntoWithFormat[interface{}]([]byte("a: 1\na: 2\n"), model.FormatYAML); err == nil {
		t.Fatal("budget off must leave duplicates rejected")
	}

	model.SetMaxYAMLKeyComparisons(-1)
	if got := model.GetMaxYAMLKeyComparisons(); got != 50_000_000 {
		t.Fatalf("negative limit gave %d", got)
	}
}

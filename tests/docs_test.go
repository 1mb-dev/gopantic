package tests

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/1mb-dev/gopantic/pkg/model"
)

// docPages returns every published page. The changelog is left out: it names tags that no
// longer work, on purpose.
func docPages(t *testing.T) map[string]string {
	t.Helper()
	paths := []string{"../README.md"}
	err := filepath.WalkDir("../docs", func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".md" && filepath.Base(path) != "changelog.md" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	pages := make(map[string]string, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		pages[path] = string(content)
	}
	return pages
}

var (
	validateTag = regexp.MustCompile(`validate:"([^"]*)"`)
	mdLink      = regexp.MustCompile(`\]\(([^)#\s]*\.md)?(#[^)\s]*)?\)`)
	mdHeading   = regexp.MustCompile(`(?m)^#+\s+(.+)$`)
)

// A tag copied from the docs has to work, because an unknown rule fails every parse. A name
// outside the built-ins must be registered on the page that uses it.
func TestDocsNameRealRules(t *testing.T) {
	builtIn := make(map[string]bool)
	for _, name := range model.NewValidatorRegistry().ListValidators() {
		builtIn[name] = true
	}

	for path, page := range docPages(t) {
		for _, match := range validateTag.FindAllStringSubmatch(page, -1) {
			for _, rule := range strings.Split(match[1], ",") {
				name, _, _ := strings.Cut(strings.TrimSpace(rule), "=")
				if name == "" || name == "-" || builtIn[name] {
					continue
				}
				if !regexp.MustCompile(`Register\w*\("` + regexp.QuoteMeta(name) + `"`).MatchString(page) {
					t.Errorf("%s: rule %q is not built in and not registered on the page", path, name)
				}
			}
		}
	}
}

func headingSlug(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(heading) {
		switch {
		case r == ' ':
			b.WriteByte('-')
		case r == '-' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestDocsLinksResolve(t *testing.T) {
	pages := docPages(t)
	for path, page := range pages {
		for _, match := range mdLink.FindAllStringSubmatch(page, -1) {
			target, anchor := path, strings.TrimPrefix(match[2], "#")
			if match[1] != "" {
				target = filepath.Join(filepath.Dir(path), match[1])
			}
			content, ok := pages[target]
			if !ok {
				existing, err := os.ReadFile(target)
				if err != nil {
					t.Errorf("%s: broken link to %s", path, match[1])
					continue
				}
				content = string(existing)
			}
			if anchor == "" {
				continue
			}
			found := false
			for _, heading := range mdHeading.FindAllStringSubmatch(content, -1) {
				found = found || headingSlug(heading[1]) == anchor
			}
			if !found {
				t.Errorf("%s: no heading for #%s in %s", path, anchor, target)
			}
		}
	}
}

// Each Behavior changes bullet names the row that holds it, so a renamed row must not leave the
// changelog pointing at nothing.
func TestChangelogNamesRealRows(t *testing.T) {
	content, err := os.ReadFile("../docs/changelog.md")
	if err != nil {
		t.Fatal(err)
	}

	rows := make(map[string]bool)
	for _, row := range contractRows() {
		rows[row.group+"/"+row.name] = true
	}

	named := regexp.MustCompile("`((?:numeric|divergence|yaml|parity|validation|tags|panics|cache|errors)/[^`]+)`")
	matches := named.FindAllStringSubmatch(string(content), -1)
	if len(matches) == 0 {
		t.Fatal("no rows named: the pattern is out of date")
	}
	for _, match := range matches {
		if !rows[match[1]] {
			t.Errorf("changelog names %q, which is not a contract row", match[1])
		}
	}
}

// Quoted output and stated behaviour, in the order the pages make the claims.

type docUser struct {
	ID    int    `json:"id" validate:"required,min=1"`
	Name  string `json:"name" validate:"required,min=2"`
	Email string `json:"email" validate:"required,email"`
	Age   int    `json:"age" validate:"min=18,max=120"`
}

type docStartUser struct {
	ID       int    `json:"id" validate:"required,min=1"`
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Age      int    `json:"age" validate:"min=0,max=150"`
}

type docPointerAge struct {
	Age *int `json:"age" validate:"required"`
}

type docTwoBytes struct {
	S string `json:"s" validate:"min=2"`
}

type docTypo struct {
	Name string `json:"name" validate:"requird"`
}

type docEven struct {
	Count int `json:"count" validate:"doc_is_even"`
}

type docPost struct {
	Status string `json:"status" validate:"doc_oneof=draft published archived"`
}

type docRegistration struct {
	Password string `json:"password" validate:"required,min=8"`
	Confirm  string `json:"confirm" validate:"required,doc_eqfield=Password"`
}

type docCreateChat struct {
	Message string `json:"message" validate:"required"`
	KBID    string `json:"-" validate:"required"`
}

type docScores struct {
	Scores []float64 `json:"scores" validate:"len=3"`
}

type docConfig struct {
	Host string `json:"host"`
	Port int    `yaml:"port" json:"p"`
	TLS  bool
}

type docNode struct {
	Name     string     `json:"name" validate:"required"`
	Children []*docNode `json:"children"`
	Parent   *docNode   `json:"-"`
}

type docCycle struct {
	Name string    `json:"name" validate:"required"`
	Next *docCycle `json:"next"`
}

func wantContains(err error, parts ...string) error {
	if err == nil {
		return errors.New("no error")
	}
	for _, part := range parts {
		if !strings.Contains(err.Error(), part) {
			return fmt.Errorf("missing %q in %q", part, err)
		}
	}
	return nil
}

func TestDocsClaims(t *testing.T) {
	model.RegisterGlobalFunc("doc_is_even", func(field string, value interface{}, _ map[string]interface{}) error {
		if n, ok := value.(int); ok && n%2 != 0 {
			return model.NewValidationError(field, value, "doc_is_even", "must be even")
		}
		return nil
	})
	model.RegisterGlobalFunc("doc_oneof", func(field string, value interface{}, params map[string]interface{}) error {
		allowed, _ := params["value"].(string)
		for _, option := range strings.Fields(allowed) {
			if value == option {
				return nil
			}
		}
		return model.NewValidationError(field, value, "doc_oneof", "must be one of: "+allowed)
	})
	model.RegisterGlobalCrossFieldFunc("doc_eqfield", func(field string, value interface{}, parent reflect.Value, params map[string]interface{}) error {
		other := parent.FieldByName(params["value"].(string))
		if !other.IsValid() || other.Interface() != value {
			return model.NewValidationError(field, value, "doc_eqfield", "fields do not match")
		}
		return nil
	})

	claims := []struct {
		name  string
		check func() error
	}{
		{"README quick start", func() error {
			got, err := model.ParseInto[docUser]([]byte(`{"id": "42", "name": "Alice", "email": "alice@example.com", "age": "28"}`))
			if err != nil || got.ID != 42 || got.Age != 28 {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"README error output", func() error {
			_, err := model.ParseInto[docUser]([]byte(`{"id": 1.5, "name": "A", "email": "nope", "age": 28}`))
			return wantContains(err,
				"multiple errors: ",
				`parse error on field "ID": value has a fractional part`,
				`validation error on field "Name": string length must be at least 2 characters`,
				`validation error on field "Email": invalid email address format`)
		}},
		{"getting started: YAML with json tags", func() error {
			got, err := model.ParseInto[docStartUser]([]byte("\nid: 456\nusername: bob\nemail: bob@example.com\nage: 30\n"))
			if err != nil || got.Username != "bob" {
				return fmt.Errorf("got %+v, err %v", got, err)
			}
			return nil
		}},
		{"getting started: error output and list", func() error {
			_, err := model.ParseInto[docStartUser]([]byte(`{"id": "x", "username": "al", "email": "alice@example.com"}`))
			if wrong := wantContains(err, `parse error on field "ID": cannot parse string "x" as integer`,
				`validation error on field "Username": string length must be at least 3 characters`); wrong != nil {
				return wrong
			}
			var list model.ErrorList
			if !errors.As(err, &list) || len(list.ValidationErrors()) != 1 {
				return fmt.Errorf("list: %v", err)
			}
			if got := list.ValidationErrors()[0]; got.FieldPath != "Username" || got.Rule != "min" {
				return fmt.Errorf("got %q %q", got.FieldPath, got.Rule)
			}
			return nil
		}},
		{"validation: pointer tells absent from zero", func() error {
			if _, err := model.ParseInto[docPointerAge]([]byte(`{}`)); err == nil {
				return errors.New("nil passed required")
			}
			_, err := model.ParseInto[docPointerAge]([]byte(`{"age": 0}`))
			return err
		}},
		{"validation: string lengths count bytes", func() error {
			_, err := model.ParseInto[docTwoBytes]([]byte(`{"s": "é"}`))
			return err
		}},
		{"validation: tag error text", func() error {
			_, err := model.ParseInto[docTypo]([]byte(`{"name": "x"}`))
			return wantContains(err, `docTypo.Name: invalid validate tag "requird": unknown validator "requird": use one of alpha, alphanum,`, "or register it first")
		}},
		{"validation: custom rule", func() error {
			if _, err := model.ParseInto[docEven]([]byte(`{"count": 3}`)); err == nil {
				return errors.New("3 passed")
			}
			_, err := model.ParseInto[docEven]([]byte(`{"count": 4}`))
			return err
		}},
		{"validation: parameter with spaces", func() error {
			if _, err := model.ParseInto[docPost]([]byte(`{"status": "gone"}`)); err == nil {
				return errors.New("gone passed")
			}
			_, err := model.ParseInto[docPost]([]byte(`{"status": "published"}`))
			return err
		}},
		{"validation: cross-field rule", func() error {
			if _, err := model.ParseInto[docRegistration]([]byte(`{"password": "abcdefgh", "confirm": "x"}`)); err == nil {
				return errors.New("mismatch passed")
			}
			_, err := model.ParseInto[docRegistration]([]byte(`{"password": "abcdefgh", "confirm": "abcdefgh"}`))
			return err
		}},
		{"validation: parse then fill in then Validate", func() error {
			req, err := model.ParseInto[docCreateChat]([]byte(`{"message": "hi"}`))
			if err != nil {
				return err
			}
			if model.Validate(&req) == nil {
				return errors.New("empty KBID passed Validate")
			}
			req.KBID = "kb1"
			return model.Validate(&req)
		}},
		{"validation: errors.As on a parse", func() error {
			_, err := model.ParseInto[docStartUser]([]byte(`{"id": 1, "username": "al", "email": "a@b.co"}`))
			var v *model.ValidationError
			if !errors.As(err, &v) || v.Rule != "min" {
				return fmt.Errorf("got %v", err)
			}
			return nil
		}},
		{"coercion: booleans", func() error {
			for input, want := range map[string]bool{"yes": true, "On": true, "1": true, "NO": false, "off": false, "": false} {
				got, err := model.CoerceValue(input, reflect.TypeOf(true), "b")
				if err != nil || got != want {
					return fmt.Errorf("%q: got %v, err %v", input, got, err)
				}
			}
			return nil
		}},
		{"coercion: map keys", func() error {
			return wantValue(`{"1": "a"}`, map[int]string{1: "a"})
		}},
		{"coercion: exact digits", func() error {
			return wantValue(`{"v": 10000000000000000}`, ctInt{V: 10000000000000000})
		}},
		{"limitations: large float into an int", func() error {
			if err := wantError[ctInt](`{"v": 9007199254740992.0}`); err != nil {
				return err
			}
			return wantError[ctInt](`{"v": 1e16}`)
		}},
		{"limitations: flow YAML needs the format named", func() error {
			if _, err := model.ParseInto[map[string]int]([]byte(`{a: 1}`)); err == nil {
				return errors.New("ParseInto accepted flow YAML")
			}
			_, err := model.ParseIntoWithFormat[map[string]int]([]byte(`{a: 1}`), model.FormatYAML)
			return err
		}},
		{"limitations: a real cycle fails Validate", func() error {
			cycle := &docCycle{Name: "a"}
			cycle.Next = cycle
			return wantContains(model.Validate(cycle), "validation depth exceeded")
		}},
		{"types: a rule requires a slice length", func() error {
			if _, err := model.ParseInto[docScores]([]byte(`{"scores": [1, 2]}`)); err == nil {
				return errors.New("two scores passed len=3")
			}
			_, err := model.ParseInto[docScores]([]byte(`{"scores": [1, 2, 3]}`))
			return err
		}},
		{"types: key resolution", func() error {
			fromYAML, err := model.ParseIntoWithFormat[docConfig]([]byte("host: h\nport: 1\ntls: true\n"), model.FormatYAML)
			if err != nil || fromYAML != (docConfig{Host: "h", Port: 1, TLS: true}) {
				return fmt.Errorf("YAML: got %+v, err %v", fromYAML, err)
			}
			fromJSON, err := model.ParseInto[docConfig]([]byte(`{"host": "h", "p": 1, "tls": true}`))
			if err != nil || fromJSON != (docConfig{Host: "h", Port: 1, TLS: true}) {
				return fmt.Errorf("JSON: got %+v, err %v", fromJSON, err)
			}
			return nil
		}},
		{"types: back-pointer tagged json:\"-\"", func() error {
			root := &docNode{Name: "root"}
			root.Children = []*docNode{{Name: "child", Parent: root}}
			return model.Validate(root)
		}},
		{"types: empty and absent slices", func() error {
			absent, err := model.ParseInto[docScores]([]byte(`{}`))
			if err != nil {
				return wantContains(err, "length must be exactly 3")
			}
			if absent.Scores != nil {
				return errors.New("absent slice is not nil")
			}
			return nil
		}},
	}

	for _, claim := range claims {
		t.Run(claim.name, func(t *testing.T) {
			if err := claim.check(); err != nil {
				t.Error(err)
			}
		})
	}
}

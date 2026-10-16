package catalog_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"rutadeorigen/lib/catalog"
)

const internalOnly = `INTERNAL: "Something went wrong"`

var genericCodes = []string{"INTERNAL", "INVALID_REQUEST_BODY", "UNAUTHORIZED", "FORBIDDEN"}

func toBytes(sources []string) [][]byte {
	out := make([][]byte, len(sources))
	for i, s := range sources {
		out[i] = []byte(s)
	}
	return out
}

func build(t *testing.T, sources ...string) *catalog.Catalog {
	t.Helper()

	c, err := catalog.New(toBytes(sources)...)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c
}

func buildWithGeneric(t *testing.T, sources ...string) *catalog.Catalog {
	t.Helper()

	c, err := catalog.NewWithGeneric(toBytes(sources)...)
	if err != nil {
		t.Fatalf("NewWithGeneric: %v", err)
	}
	return c
}

func TestNewMergesSources(t *testing.T) {
	c := build(t, internalOnly, `LOT_VOIDED: "The lot is voided"`, `FARM_NOT_FOUND: "No farm was found"`)

	for _, code := range []string{"INTERNAL", "LOT_VOIDED", "FARM_NOT_FOUND"} {
		if !c.Has(code) {
			t.Errorf("catalog is missing %q", code)
		}
	}
}

func TestNewFails(t *testing.T) {
	cases := map[string]struct {
		sources []string
		wantErr string
	}{
		"no sources": {
			sources: nil,
			wantErr: "INTERNAL",
		},
		"missing INTERNAL": {
			sources: []string{`LOT_VOIDED: "The lot is voided"`},
			wantErr: "missing required code",
		},
		"INTERNAL with parameters": {
			sources: []string{`INTERNAL: "Failed for {reason}"`},
			wantErr: "must not use parameters",
		},
		"duplicate code across sources": {
			sources: []string{internalOnly, `LOT_VOIDED: "first"`, `LOT_VOIDED: "second"`},
			wantErr: "duplicate code",
		},
		"invalid yaml": {
			sources: []string{internalOnly, `LOT_VOIDED: [unclosed`},
			wantErr: "invalid yaml",
		},
		"empty source": {
			sources: []string{internalOnly, ``},
			wantErr: "no messages defined",
		},
		"empty message": {
			sources: []string{internalOnly, `LOT_VOIDED: ""`},
			wantErr: "empty message",
		},
		"blank message": {
			sources: []string{internalOnly, `LOT_VOIDED: "   "`},
			wantErr: "empty message",
		},
		"unclosed placeholder": {
			sources: []string{internalOnly, `LOT_VOIDED: "Lot {lot_id is voided"`},
			wantErr: "unclosed",
		},
		"stray closing brace": {
			sources: []string{internalOnly, `LOT_VOIDED: "Lot is voided}"`},
			wantErr: "unexpected",
		},
		"empty placeholder": {
			sources: []string{internalOnly, `LOT_VOIDED: "Lot {} is voided"`},
			wantErr: "invalid placeholder",
		},
		"nested placeholder": {
			sources: []string{internalOnly, `LOT_VOIDED: "Lot {a{b}} is voided"`},
			wantErr: "invalid placeholder",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c, err := catalog.New(toBytes(tc.sources)...)
			if err == nil {
				t.Fatalf("expected an error, got catalog %v", c)
			}
			if c != nil {
				t.Errorf("catalog must be nil on error, got %v", c)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestNewReportsEverySourceProblem(t *testing.T) {
	_, err := catalog.New(
		[]byte(internalOnly),
		[]byte(`LOT_VOIDED: ""`),
		[]byte(`FARM_NOT_FOUND: "Farm {id"`),
	)
	if err == nil {
		t.Fatal("expected an error")
	}

	for _, want := range []string{"source 1", "source 2"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestResolve(t *testing.T) {
	c := build(t, internalOnly,
		`PLAIN: "The lot is voided"`,
		`RANGE: "The code must have between {min} and {max} characters"`,
		`REPEATED: "{value} and {value}"`,
		`ADJACENT: "{a}{b}"`,
		`PADDED: "   trimmed   "`,
		`ONLY_PARAM: "{value}"`,
	)

	cases := map[string]struct {
		code   string
		params map[string]any
		want   string
	}{
		"plain message":              {"PLAIN", nil, "The lot is voided"},
		"plain message empty params": {"PLAIN", map[string]any{}, "The lot is voided"},
		"int parameters":             {"RANGE", map[string]any{"min": 3, "max": 30}, "The code must have between 3 and 30 characters"},
		"string parameters":          {"RANGE", map[string]any{"min": "three", "max": "thirty"}, "The code must have between three and thirty characters"},
		"extra parameter ignored":    {"PLAIN", map[string]any{"unused": 1}, "The lot is voided"},
		"repeated parameter":         {"REPEATED", map[string]any{"value": 7}, "7 and 7"},
		"adjacent parameters":        {"ADJACENT", map[string]any{"a": 1, "b": 2}, "12"},
		"surrounding spaces trimmed": {"PADDED", nil, "trimmed"},
		"message that is a param":    {"ONLY_PARAM", map[string]any{"value": 1.5}, "1.5"},
		"bool parameter":             {"ONLY_PARAM", map[string]any{"value": true}, "true"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := c.Resolve(tc.code, tc.params)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveUnknownCodeFallsBackToInternal(t *testing.T) {
	c := build(t, internalOnly)

	got, err := c.Resolve("NOT_A_REAL_CODE", nil)

	if !errors.Is(err, catalog.ErrUnknownCode) {
		t.Fatalf("error: got %v, want ErrUnknownCode", err)
	}
	if !strings.Contains(err.Error(), "NOT_A_REAL_CODE") {
		t.Errorf("error %q must name the missing code", err)
	}
	if got != "Something went wrong" {
		t.Errorf("got %q, want the INTERNAL message", got)
	}
}

func TestResolveMissingParameterFallsBackToInternal(t *testing.T) {
	c := build(t, internalOnly, `RANGE: "Between {min} and {max}"`)

	cases := map[string]map[string]any{
		"nil params":       nil,
		"empty params":     {},
		"one param absent": {"min": 3},
	}

	for name, params := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := c.Resolve("RANGE", params)

			if !errors.Is(err, catalog.ErrMissingParam) {
				t.Errorf("error: got %v, want ErrMissingParam", err)
			}
			if got != "Something went wrong" {
				t.Errorf("got %q, want the INTERNAL message", got)
			}
			if strings.Contains(got, "{") {
				t.Errorf("message %q leaks a placeholder", got)
			}
		})
	}
}

func TestResolveNeverReturnsEmptyMessage(t *testing.T) {
	c := build(t, internalOnly, `RANGE: "Between {min} and {max}"`)

	inputs := []struct {
		code   string
		params map[string]any
	}{
		{"INTERNAL", nil},
		{"RANGE", nil},
		{"RANGE", map[string]any{"min": 1, "max": 2}},
		{"UNKNOWN", nil},
		{"", nil},
	}

	for _, in := range inputs {
		got, _ := c.Resolve(in.code, in.params)
		if strings.TrimSpace(got) == "" {
			t.Errorf("Resolve(%q, %v) returned an empty message", in.code, in.params)
		}
	}
}

func TestResolveDoesNotModifyParams(t *testing.T) {
	c := build(t, internalOnly, `RANGE: "Between {min} and {max}"`)
	params := map[string]any{"min": 3, "max": 30}

	if _, err := c.Resolve("RANGE", params); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(params) != 2 || params["min"] != 3 || params["max"] != 30 {
		t.Errorf("params were modified: %v", params)
	}
}

func TestHas(t *testing.T) {
	c := build(t, internalOnly, `LOT_VOIDED: "The lot is voided"`)

	if !c.Has("LOT_VOIDED") {
		t.Error("Has(LOT_VOIDED) = false, want true")
	}
	if c.Has("FARM_NOT_FOUND") {
		t.Error("Has(FARM_NOT_FOUND) = true, want false")
	}
	if c.Has("") {
		t.Error("Has(\"\") = true, want false")
	}
}

func TestCodes(t *testing.T) {
	c := build(t, internalOnly, `LOT_VOIDED: "The lot is voided"`, `FARM_NOT_FOUND: "No farm was found"`)

	got := c.Codes()
	slices.Sort(got)

	want := []string{"FARM_NOT_FOUND", "INTERNAL", "LOT_VOIDED"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCodesReturnsIndependentSlice(t *testing.T) {
	c := build(t, internalOnly, `LOT_VOIDED: "The lot is voided"`)

	first := c.Codes()
	for i := range first {
		first[i] = "CHANGED"
	}

	if !c.Has("LOT_VOIDED") || !c.Has("INTERNAL") {
		t.Error("modifying the result of Codes changed the catalog")
	}
}

func TestNewWithGenericIncludesGenericCodes(t *testing.T) {
	c := buildWithGeneric(t)

	for _, code := range genericCodes {
		if !c.Has(code) {
			t.Errorf("generic catalog is missing %q", code)
		}
	}
}

func TestGenericMessagesResolveWithoutParams(t *testing.T) {
	c := buildWithGeneric(t)

	for _, code := range genericCodes {
		got, err := c.Resolve(code, nil)
		if err != nil {
			t.Errorf("Resolve(%q): %v", code, err)
		}
		if strings.TrimSpace(got) == "" {
			t.Errorf("generic message for %q is empty", code)
		}
	}
}

func TestNewWithGenericMergesServiceSources(t *testing.T) {
	c := buildWithGeneric(t, `LOT_VOIDED: "The lot is voided"`, `FARM_NOT_FOUND: "No farm was found"`)

	for _, code := range append([]string{"LOT_VOIDED", "FARM_NOT_FOUND"}, genericCodes...) {
		if !c.Has(code) {
			t.Errorf("catalog is missing %q", code)
		}
	}

	got, err := c.Resolve("LOT_VOIDED", nil)
	if err != nil || got != "The lot is voided" {
		t.Errorf("Resolve(LOT_VOIDED) = %q, %v", got, err)
	}
}

func TestNewWithGenericRejectsServiceRedefiningGenericCode(t *testing.T) {
	for _, code := range genericCodes {
		t.Run(code, func(t *testing.T) {
			_, err := catalog.NewWithGeneric([]byte(code + `: "Redefined by the service"`))

			if err == nil {
				t.Fatal("expected a duplicate code error")
			}
			if !strings.Contains(err.Error(), "duplicate code") {
				t.Errorf("error %q does not report a duplicate code", err)
			}
		})
	}
}

func TestNewWithGenericIsRepeatable(t *testing.T) {
	service := []byte(`LOT_VOIDED: "The lot is voided"`)

	for i := 0; i < 3; i++ {
		c, err := catalog.NewWithGeneric(service)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if !c.Has("LOT_VOIDED") || !c.Has("INTERNAL") {
			t.Errorf("call %d: catalog is incomplete", i)
		}
	}
}

func TestNewWithGenericFailsOnInvalidServiceSource(t *testing.T) {
	_, err := catalog.NewWithGeneric([]byte(`LOT_VOIDED: "Lot {id is voided"`))

	if err == nil {
		t.Fatal("expected an error")
	}
}

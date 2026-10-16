package catalog

import (
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

const internalCode = "INTERNAL"

var (
	ErrUnknownCode  = errors.New("unknown message code")
	ErrMissingParam = errors.New("missing message parameter")
)

//go:embed messages.yaml
var genericYAML []byte

type token struct {
	text    string
	isParam bool
}

type Catalog struct {
	messages map[string][]token
	fallback string
}

func NewWithGeneric(sources ...[]byte) (*Catalog, error) {
	all := make([][]byte, 0, len(sources)+1)
	all = append(all, genericYAML)
	all = append(all, sources...)

	return NewFromSources(all...)
}

func NewFromSources(sources ...[]byte) (*Catalog, error) {
	messages := make(map[string][]token)
	var errs []error

	for i, src := range sources {
		parsed, err := parse(src)
		if err != nil {
			errs = append(errs, fmt.Errorf("source %d: %w", i, err))
			continue
		}
		for code, tokens := range parsed {
			if _, dup := messages[code]; dup {
				errs = append(errs, fmt.Errorf("source %d: duplicate code %q", i, code))
				continue
			}
			messages[code] = tokens
		}
	}

	fallback, err := internalMessage(messages)
	if err != nil {
		errs = append(errs, err)
	}

	if joined := errors.Join(errs...); joined != nil {
		return nil, fmt.Errorf("catalog: %w", joined)
	}

	return &Catalog{messages: messages, fallback: fallback}, nil
}

func (c *Catalog) Resolve(code string, params map[string]any) (string, error) {
	tokens, ok := c.messages[code]
	if !ok {
		return c.fallback, fmt.Errorf("%w: %q", ErrUnknownCode, code)
	}

	msg, err := render(tokens, params)
	if err != nil {
		return c.fallback, fmt.Errorf("code %q: %w", code, err)
	}

	return msg, nil
}
func (c *Catalog) Has(code string) bool {
	_, ok := c.messages[code]
	return ok
}

func (c *Catalog) Codes() []string {
	codes := make([]string, 0, len(c.messages))
	for code := range c.messages {
		codes = append(codes, code)
	}
	return codes
}

func internalMessage(messages map[string][]token) (string, error) {
	tokens, ok := messages[internalCode]
	if !ok {
		return "", fmt.Errorf("missing required code %q", internalCode)
	}

	var b strings.Builder
	for _, t := range tokens {
		if t.isParam {
			return "", fmt.Errorf("code %q must not use parameters", internalCode)
		}
		b.WriteString(t.text)
	}

	return b.String(), nil
}

func parse(src []byte) (map[string][]token, error) {
	var raw map[string]string
	if err := yaml.Unmarshal(src, &raw); err != nil {
		return nil, fmt.Errorf("invalid yaml: %w", err)
	}
	if len(raw) == 0 {
		return nil, errors.New("no messages defined")
	}

	parsed := make(map[string][]token, len(raw))
	var errs []error

	for code, msg := range raw {
		if strings.TrimSpace(code) == "" {
			errs = append(errs, errors.New("empty code"))
			continue
		}
		msg = strings.TrimSpace(msg)
		if msg == "" {
			errs = append(errs, fmt.Errorf("code %q: empty message", code))
			continue
		}
		tokens, err := tokenize(msg)
		if err != nil {
			errs = append(errs, fmt.Errorf("code %q: %w", code, err))
			continue
		}
		parsed[code] = tokens
	}

	if joined := errors.Join(errs...); joined != nil {
		return nil, joined
	}

	return parsed, nil
}

func tokenize(msg string) ([]token, error) {
	var tokens []token

	for len(msg) > 0 {
		i := strings.IndexAny(msg, "{}")
		if i < 0 {
			tokens = append(tokens, token{text: msg})
			break
		}
		if msg[i] == '}' {
			return nil, fmt.Errorf("unexpected '}' in %q", msg)
		}

		end := strings.IndexByte(msg[i:], '}')
		if end < 0 {
			return nil, fmt.Errorf("unclosed '{' in %q", msg)
		}

		name := msg[i+1 : i+end]
		if name == "" || strings.Contains(name, "{") {
			return nil, fmt.Errorf("invalid placeholder %q", msg[i:i+end+1])
		}

		if i > 0 {
			tokens = append(tokens, token{text: msg[:i]})
		}
		tokens = append(tokens, token{text: name, isParam: true})
		msg = msg[i+end+1:]
	}

	return tokens, nil
}

func render(tokens []token, params map[string]any) (string, error) {
	var b strings.Builder

	for _, t := range tokens {
		if !t.isParam {
			b.WriteString(t.text)
			continue
		}
		v, ok := params[t.text]
		if !ok {
			return "", fmt.Errorf("%w: %q", ErrMissingParam, t.text)
		}
		_, _ = fmt.Fprint(&b, v)
	}

	return b.String(), nil
}

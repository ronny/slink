package ids

import (
	"fmt"

	"github.com/jaevor/go-nanoid"
)

const (
	// The alphabet that can make up an ID, every char should be safe for use in URLs without extra encoding
	NanoIDDefaultCharacters = "01234567890ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-"
	NanoIDDefaultLength     = 10
	// the default max number of attempts to generate an ID until it doesn't
	// match anything in the denylist
	NanoIDDefaultMaxAttempts = 10
)

type NanoIDGenerator struct {
	chars       string
	length      int
	maxAttempts int
	denylist    []string
	generate    func() string
}

func (g *NanoIDGenerator) GenerateID() (string, error) {
	for attempt := 0; attempt < g.maxAttempts; attempt++ {
		id := g.generate()
		if IsAllowed(id, g.denylist) {
			return id, nil
		}
	}

	return "", fmt.Errorf("exhausted %d attempts to generate an ID that doesn't match anything from the denylist", g.maxAttempts)
}

func NewNanoIDGenerator(options ...func(*NanoIDGenerator)) (*NanoIDGenerator, error) {
	generator := &NanoIDGenerator{
		chars:       NanoIDDefaultCharacters,
		length:      NanoIDDefaultLength,
		maxAttempts: NanoIDDefaultMaxAttempts,
	}

	for _, option := range options {
		option(generator)
	}

	// An empty, non-nil denylist with a length of 0 indicates the user wants no
	// denylist. Only load the default denylist if this is not nil.
	if generator.denylist == nil {
		generator.denylist = defaultDenylist
	}

	var err error
	generator.generate, err = nanoid.CustomASCII(generator.chars, generator.length)
	if err != nil {
		return nil, fmt.Errorf("nanoid.CustomASCII: %w", err)
	}

	return generator, nil
}

func WithNanoIDCustomASCII(chars string) func(*NanoIDGenerator) {
	return func(g *NanoIDGenerator) {
		g.chars = chars
	}
}

// WithNanoIDLength specifies a specific length for the generated Nano IDs.
// See https://zelark.github.io/nano-id-cc/ on length and collisions.
func WithNanoIDLength(length int) func(*NanoIDGenerator) {
	return func(g *NanoIDGenerator) {
		g.length = length
	}
}

// WithNanoIDLength specifies the max number of attempts to generate an ID until
// it doesn't match anything in the denylist
func WithNanoIDMaxAttempts(maxAttempts int) func(*NanoIDGenerator) {
	return func(g *NanoIDGenerator) {
		g.maxAttempts = maxAttempts
	}
}

func WithNanoIDDenylist(denylist []string) func(*NanoIDGenerator) {
	return func(g *NanoIDGenerator) {
		g.denylist = denylist
	}
}

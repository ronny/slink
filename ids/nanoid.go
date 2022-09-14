package ids

import (
	"fmt"

	"github.com/jaevor/go-nanoid"
)

const (
	NanoIDDefaultCharacters = "01234567890ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	NanoIDDefaultLength     = 10
)

type NanoIDGenerator struct {
	chars    string
	length   int
	generate func() string
}

func NewNanoIDGenerator(options ...func(*NanoIDGenerator)) (*NanoIDGenerator, error) {
	generator := &NanoIDGenerator{
		chars:  NanoIDDefaultCharacters,
		length: NanoIDDefaultLength,
	}

	for _, option := range options {
		option(generator)
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

func (g *NanoIDGenerator) GenerateID() string {
	return g.generate()
}

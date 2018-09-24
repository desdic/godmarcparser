package input

import (
	"context"
	"dmarc/pkg/dmarc"
)

// Handler is the interface for input handling of file types
type Handler interface {
	Read(ctx context.Context, input string, queue chan<- dmarc.Content) error
}

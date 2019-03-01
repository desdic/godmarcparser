package storage

import (
	"context"

	"github.com/desdic/godmarcparser/dmarc"
)

// Storage is the interface type so we can use different drivers
// TODO: The name storage.Storage is redundant
type Storage interface {
	Initialize(ctx context.Context) error
	Write(ctx context.Context, f dmarc.Feedback) error
	ReadReports(ctx context.Context, offset int, pagesize int) ([]dmarc.Report, error)
	ReadReport(ctx context.Context, id int64) (dmarc.Rows, error)
}

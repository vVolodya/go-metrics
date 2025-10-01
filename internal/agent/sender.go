package agent

import "context"

type Sender interface {
	ReportOnce(ctx context.Context) error
}

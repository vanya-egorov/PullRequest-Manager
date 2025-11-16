package stats

import (
	"context"

	"github.com/vanya-egorov/PullRequest-Manager/internal/entities"
)

type StatsUseCase interface {
	GetStats(ctx context.Context) (entities.Stats, error)
}

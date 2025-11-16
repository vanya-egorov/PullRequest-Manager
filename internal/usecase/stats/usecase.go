package stats

import (
	"context"

	"github.com/vanya-egorov/PullRequest-Manager/internal/entities"
	"github.com/vanya-egorov/PullRequest-Manager/internal/repository"
	"github.com/vanya-egorov/PullRequest-Manager/pkg/logger"
)

type useCase struct {
	statsRepo repository.StatsRepository
	logger    logger.Logger
}

func New(statsRepo repository.StatsRepository, log logger.Logger) StatsUseCase {
	return &useCase{
		statsRepo: statsRepo,
		logger:    log,
	}
}

func (u *useCase) GetStats(ctx context.Context) (entities.Stats, error) {
	assignments, err := u.statsRepo.ListReviewerAssignments(ctx)
	if err != nil {
		return entities.Stats{}, err
	}

	open, err := u.statsRepo.CountOpenPullRequests(ctx)
	if err != nil {
		return entities.Stats{}, err
	}

	return entities.Stats{
		AssignmentsByUser: assignments,
		OpenPRs:           open,
	}, nil
}

package usecase

import (
	"context"

	"github.com/vanya-egorov/PullRequest-Manager/internal/entities"
)

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

package usecase

import (
	"context"

	"github.com/vanya-egorov/PullRequest-Manager/internal/entities"
)

type PullRequestUseCase interface {
	CreatePullRequest(ctx context.Context, input CreatePullRequestInput) (entities.PullRequest, error)
	MergePullRequest(ctx context.Context, prID string) (entities.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID string, oldUserID string) (ReassignResult, error)
	GetUserReviews(ctx context.Context, userID string) ([]entities.PullRequestShort, error)
}

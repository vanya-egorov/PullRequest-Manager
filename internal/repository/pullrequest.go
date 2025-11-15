package repository

import (
	"context"

	"github.com/vanya-egorov/PullRequest-Manager/internal/entities"
)

type PullRequestRepository interface {
	CreatePullRequest(ctx context.Context, pr entities.PullRequest) (entities.PullRequest, error)
	GetPullRequest(ctx context.Context, prID string) (entities.PullRequest, error)
	SetPullRequestStatusMerged(ctx context.Context, prID string) (entities.PullRequest, error)
	ListAssignedReviewers(ctx context.Context, prID string) ([]string, error)
	ReplaceReviewer(ctx context.Context, prID string, oldUserID string, newUserID *string) error
	ListReviewPullRequests(ctx context.Context, userID string) ([]entities.PullRequestShort, error)
	UpdateNeedMoreReviewers(ctx context.Context, prID string, need bool) error
	ListOpenPullRequestsByReviewers(ctx context.Context, userIDs []string) (map[string][]entities.PullRequest, error)
}

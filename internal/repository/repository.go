package repository

import (
	"context"

	"github.com/vanya-egorov/PullRequest-Manager/internal/entities"
)

type Repository interface {
	CreateTeam(ctx context.Context, name string, members []entities.TeamMember) (entities.Team, error)
	GetTeam(ctx context.Context, name string) (entities.Team, error)
	GetUser(ctx context.Context, userID string) (entities.User, error)
	SetUserActive(ctx context.Context, userID string, isActive bool) (entities.User, error)
	ListUsersByTeam(ctx context.Context, teamName string, onlyActive bool) ([]entities.User, error)
	CreatePullRequest(ctx context.Context, pr entities.PullRequest) (entities.PullRequest, error)
	GetPullRequest(ctx context.Context, prID string) (entities.PullRequest, error)
	SetPullRequestStatusMerged(ctx context.Context, prID string) (entities.PullRequest, error)
	ListAssignedReviewers(ctx context.Context, prID string) ([]string, error)
	ReplaceReviewer(ctx context.Context, prID string, oldUserID string, newUserID *string) error
	ListReviewPullRequests(ctx context.Context, userID string) ([]entities.PullRequestShort, error)
	ListReviewerAssignments(ctx context.Context) (map[string]int, error)
	CountOpenPullRequests(ctx context.Context) (int, error)
	UpdateNeedMoreReviewers(ctx context.Context, prID string, need bool) error
	ListOpenPullRequestsByReviewers(ctx context.Context, userIDs []string) (map[string][]entities.PullRequest, error)
	BulkSetUsersActive(ctx context.Context, teamName string, userIDs []string, isActive bool) ([]entities.User, error)
}

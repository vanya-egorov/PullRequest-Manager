package repository

import "context"

type StatsRepository interface {
	ListReviewerAssignments(ctx context.Context) (map[string]int, error)
	CountOpenPullRequests(ctx context.Context) (int, error)
}

package stats

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vanya-egorov/PullRequest-Manager/pkg/logger"
)

type mockStatsRepo struct {
	listReviewerAssignments func(ctx context.Context) (map[string]int, error)
	countOpenPullRequests   func(ctx context.Context) (int, error)
}

func (m *mockStatsRepo) ListReviewerAssignments(ctx context.Context) (map[string]int, error) {
	if m.listReviewerAssignments != nil {
		return m.listReviewerAssignments(ctx)
	}
	return map[string]int{}, nil
}

func (m *mockStatsRepo) CountOpenPullRequests(ctx context.Context) (int, error) {
	if m.countOpenPullRequests != nil {
		return m.countOpenPullRequests(ctx)
	}
	return 0, nil
}

func TestUseCase_GetStats(t *testing.T) {
	repo := &mockStatsRepo{
		listReviewerAssignments: func(ctx context.Context) (map[string]int, error) { return map[string]int{"ivan": 5}, nil },
		countOpenPullRequests:   func(ctx context.Context) (int, error) { return 10, nil },
	}
	uc := New(repo, logger.New())
	result, err := uc.GetStats(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 10, result.OpenPRs)
	assert.Equal(t, 5, result.AssignmentsByUser["ivan"])
}

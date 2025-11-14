package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vanya-egorov/PullRequest-Manager/internal/entities"
	"github.com/vanya-egorov/PullRequest-Manager/pkg/logger"
)

type mockRepository struct {
	createTeamFunc                      func(ctx context.Context, name string, members []entities.TeamMember) (entities.Team, error)
	getTeamFunc                         func(ctx context.Context, name string) (entities.Team, error)
	getUserFunc                         func(ctx context.Context, userID string) (entities.User, error)
	setUserActiveFunc                   func(ctx context.Context, userID string, isActive bool) (entities.User, error)
	listUsersByTeamFunc                 func(ctx context.Context, teamName string, onlyActive bool) ([]entities.User, error)
	createPullRequestFunc               func(ctx context.Context, pr entities.PullRequest) (entities.PullRequest, error)
	getPullRequestFunc                  func(ctx context.Context, prID string) (entities.PullRequest, error)
	setPullRequestStatusMergedFunc      func(ctx context.Context, prID string) (entities.PullRequest, error)
	listAssignedReviewersFunc           func(ctx context.Context, prID string) ([]string, error)
	replaceReviewerFunc                 func(ctx context.Context, prID string, oldUserID string, newUserID *string) error
	listReviewPullRequestsFunc          func(ctx context.Context, userID string) ([]entities.PullRequestShort, error)
	listReviewerAssignmentsFunc         func(ctx context.Context) (map[string]int, error)
	countOpenPullRequestsFunc           func(ctx context.Context) (int, error)
	updateNeedMoreReviewersFunc         func(ctx context.Context, prID string, need bool) error
	listOpenPullRequestsByReviewersFunc func(ctx context.Context, userIDs []string) (map[string][]entities.PullRequest, error)
	bulkSetUsersActiveFunc              func(ctx context.Context, teamName string, userIDs []string, isActive bool) ([]entities.User, error)
}

func (m *mockRepository) CreateTeam(ctx context.Context, name string, members []entities.TeamMember) (entities.Team, error) {
	return m.createTeamFunc(ctx, name, members)
}

func (m *mockRepository) GetTeam(ctx context.Context, name string) (entities.Team, error) {
	return m.getTeamFunc(ctx, name)
}

func (m *mockRepository) GetUser(ctx context.Context, userID string) (entities.User, error) {
	return m.getUserFunc(ctx, userID)
}

func (m *mockRepository) SetUserActive(ctx context.Context, userID string, isActive bool) (entities.User, error) {
	return m.setUserActiveFunc(ctx, userID, isActive)
}

func (m *mockRepository) ListUsersByTeam(ctx context.Context, teamName string, onlyActive bool) ([]entities.User, error) {
	return m.listUsersByTeamFunc(ctx, teamName, onlyActive)
}

func (m *mockRepository) CreatePullRequest(ctx context.Context, pr entities.PullRequest) (entities.PullRequest, error) {
	return m.createPullRequestFunc(ctx, pr)
}

func (m *mockRepository) GetPullRequest(ctx context.Context, prID string) (entities.PullRequest, error) {
	return m.getPullRequestFunc(ctx, prID)
}

func (m *mockRepository) SetPullRequestStatusMerged(ctx context.Context, prID string) (entities.PullRequest, error) {
	return m.setPullRequestStatusMergedFunc(ctx, prID)
}

func (m *mockRepository) ListAssignedReviewers(ctx context.Context, prID string) ([]string, error) {
	return m.listAssignedReviewersFunc(ctx, prID)
}

func (m *mockRepository) ReplaceReviewer(ctx context.Context, prID string, oldUserID string, newUserID *string) error {
	return m.replaceReviewerFunc(ctx, prID, oldUserID, newUserID)
}

func (m *mockRepository) ListReviewPullRequests(ctx context.Context, userID string) ([]entities.PullRequestShort, error) {
	return m.listReviewPullRequestsFunc(ctx, userID)
}

func (m *mockRepository) ListReviewerAssignments(ctx context.Context) (map[string]int, error) {
	return m.listReviewerAssignmentsFunc(ctx)
}

func (m *mockRepository) CountOpenPullRequests(ctx context.Context) (int, error) {
	return m.countOpenPullRequestsFunc(ctx)
}

func (m *mockRepository) UpdateNeedMoreReviewers(ctx context.Context, prID string, need bool) error {
	return m.updateNeedMoreReviewersFunc(ctx, prID, need)
}

func (m *mockRepository) ListOpenPullRequestsByReviewers(ctx context.Context, userIDs []string) (map[string][]entities.PullRequest, error) {
	return m.listOpenPullRequestsByReviewersFunc(ctx, userIDs)
}

func (m *mockRepository) BulkSetUsersActive(ctx context.Context, teamName string, userIDs []string, isActive bool) ([]entities.User, error) {
	return m.bulkSetUsersActiveFunc(ctx, teamName, userIDs, isActive)
}

func TestUseCase_CreateTeam(t *testing.T) {
	log := logger.New()

	t.Run("success", func(t *testing.T) {
		team := entities.Team{
			Name: "backend",
			Members: []entities.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
		}
		mockRepo := &mockRepository{
			createTeamFunc: func(ctx context.Context, name string, members []entities.TeamMember) (entities.Team, error) {
				return team, nil
			},
		}
		uc := New(mockRepo, log)

		result, err := uc.CreateTeam(context.Background(), team)
		assert.NoError(t, err)
		assert.Equal(t, "backend", result.Name)
	})

	t.Run("empty name", func(t *testing.T) {
		mockRepo := &mockRepository{}
		uc := New(mockRepo, log)
		team := entities.Team{Name: ""}
		_, err := uc.CreateTeam(context.Background(), team)
		assert.Error(t, err)
	})
}

func TestUseCase_CreatePullRequest(t *testing.T) {
	log := logger.New()

	t.Run("success", func(t *testing.T) {
		author := entities.User{
			ID:       "u1",
			Username: "Alice",
			TeamName: "backend",
			IsActive: true,
		}
		members := []entities.User{
			{ID: "u2", Username: "Bob", TeamName: "backend", IsActive: true},
			{ID: "u3", Username: "Carol", TeamName: "backend", IsActive: true},
		}

		mockRepo := &mockRepository{
			getUserFunc: func(ctx context.Context, userID string) (entities.User, error) {
				return author, nil
			},
			listUsersByTeamFunc: func(ctx context.Context, teamName string, onlyActive bool) ([]entities.User, error) {
				return members, nil
			},
			createPullRequestFunc: func(ctx context.Context, pr entities.PullRequest) (entities.PullRequest, error) {
				return pr, nil
			},
		}
		uc := New(mockRepo, log)

		input := CreatePullRequestInput{
			ID:       "pr-1",
			Name:     "Feature",
			AuthorID: "u1",
		}
		result, err := uc.CreatePullRequest(context.Background(), input)
		assert.NoError(t, err)
		assert.Equal(t, "pr-1", result.ID)
	})

	t.Run("author not found", func(t *testing.T) {
		mockRepo := &mockRepository{
			getUserFunc: func(ctx context.Context, userID string) (entities.User, error) {
				return entities.User{}, entities.ErrUserNotFound
			},
		}
		uc := New(mockRepo, log)

		input := CreatePullRequestInput{
			ID:       "pr-1",
			Name:     "Feature",
			AuthorID: "u1",
		}
		_, err := uc.CreatePullRequest(context.Background(), input)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, entities.ErrAuthorNotFound))
	})
}

func TestUseCase_ReassignReviewer(t *testing.T) {
	log := logger.New()

	t.Run("pr merged", func(t *testing.T) {
		pr := entities.PullRequest{
			ID:     "pr-1",
			Status: entities.StatusMerged,
		}
		mockRepo := &mockRepository{
			getPullRequestFunc: func(ctx context.Context, prID string) (entities.PullRequest, error) {
				return pr, nil
			},
		}
		uc := New(mockRepo, log)

		_, err := uc.ReassignReviewer(context.Background(), "pr-1", "u2")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, entities.ErrPullRequestMerged))
	})
}

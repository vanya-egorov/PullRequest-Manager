package team

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vanya-egorov/PullRequest-Manager/internal/entities"
	"github.com/vanya-egorov/PullRequest-Manager/pkg/logger"
)

type mockTeamRepo struct {
	createTeam         func(ctx context.Context, name string, members []entities.TeamMember) (entities.Team, error)
	getTeam            func(ctx context.Context, name string) (entities.Team, error)
	getUser            func(ctx context.Context, userID string) (entities.User, error)
	setUserActive      func(ctx context.Context, userID string, isActive bool) (entities.User, error)
	listUsersByTeam    func(ctx context.Context, teamName string, onlyActive bool) ([]entities.User, error)
	bulkSetUsersActive func(ctx context.Context, teamName string, userIDs []string, isActive bool) ([]entities.User, error)
}

func (m *mockTeamRepo) CreateTeam(ctx context.Context, name string, members []entities.TeamMember) (entities.Team, error) {
	if m.createTeam != nil {
		return m.createTeam(ctx, name, members)
	}
	return entities.Team{}, nil
}

func (m *mockTeamRepo) GetTeam(ctx context.Context, name string) (entities.Team, error) {
	if m.getTeam != nil {
		return m.getTeam(ctx, name)
	}
	return entities.Team{}, nil
}

func (m *mockTeamRepo) GetUser(ctx context.Context, userID string) (entities.User, error) {
	if m.getUser != nil {
		return m.getUser(ctx, userID)
	}
	return entities.User{}, nil
}

func (m *mockTeamRepo) SetUserActive(ctx context.Context, userID string, isActive bool) (entities.User, error) {
	if m.setUserActive != nil {
		return m.setUserActive(ctx, userID, isActive)
	}
	return entities.User{}, nil
}

func (m *mockTeamRepo) ListUsersByTeam(ctx context.Context, teamName string, onlyActive bool) ([]entities.User, error) {
	if m.listUsersByTeam != nil {
		return m.listUsersByTeam(ctx, teamName, onlyActive)
	}
	return []entities.User{}, nil
}

func (m *mockTeamRepo) BulkSetUsersActive(ctx context.Context, teamName string, userIDs []string, isActive bool) ([]entities.User, error) {
	if m.bulkSetUsersActive != nil {
		return m.bulkSetUsersActive(ctx, teamName, userIDs, isActive)
	}
	return []entities.User{}, nil
}

type mockPullRequestRepo struct {
	createPullRequest               func(ctx context.Context, pr entities.PullRequest) (entities.PullRequest, error)
	getPullRequest                  func(ctx context.Context, prID string) (entities.PullRequest, error)
	setPullRequestStatusMerged      func(ctx context.Context, prID string) (entities.PullRequest, error)
	listAssignedReviewers           func(ctx context.Context, prID string) ([]string, error)
	replaceReviewer                 func(ctx context.Context, prID string, oldUserID string, newUserID *string) error
	listReviewPullRequests          func(ctx context.Context, userID string) ([]entities.PullRequestShort, error)
	updateNeedMoreReviewers         func(ctx context.Context, prID string, need bool) error
	listOpenPullRequestsByReviewers func(ctx context.Context, userIDs []string) (map[string][]entities.PullRequest, error)
}

func (m *mockPullRequestRepo) CreatePullRequest(ctx context.Context, pr entities.PullRequest) (entities.PullRequest, error) {
	if m.createPullRequest != nil {
		return m.createPullRequest(ctx, pr)
	}
	return pr, nil
}

func (m *mockPullRequestRepo) SetPullRequestStatusMerged(ctx context.Context, prID string) (entities.PullRequest, error) {
	if m.setPullRequestStatusMerged != nil {
		return m.setPullRequestStatusMerged(ctx, prID)
	}
	return entities.PullRequest{}, nil
}

func (m *mockPullRequestRepo) ListReviewPullRequests(ctx context.Context, userID string) ([]entities.PullRequestShort, error) {
	if m.listReviewPullRequests != nil {
		return m.listReviewPullRequests(ctx, userID)
	}
	return []entities.PullRequestShort{}, nil
}

func (m *mockPullRequestRepo) ListOpenPullRequestsByReviewers(ctx context.Context, userIDs []string) (map[string][]entities.PullRequest, error) {
	if m.listOpenPullRequestsByReviewers != nil {
		return m.listOpenPullRequestsByReviewers(ctx, userIDs)
	}
	return map[string][]entities.PullRequest{}, nil
}

func (m *mockPullRequestRepo) ListAssignedReviewers(ctx context.Context, prID string) ([]string, error) {
	if m.listAssignedReviewers != nil {
		return m.listAssignedReviewers(ctx, prID)
	}
	return []string{}, nil
}

func (m *mockPullRequestRepo) ReplaceReviewer(ctx context.Context, prID string, oldUserID string, newUserID *string) error {
	if m.replaceReviewer != nil {
		return m.replaceReviewer(ctx, prID, oldUserID, newUserID)
	}
	return nil
}

func (m *mockPullRequestRepo) UpdateNeedMoreReviewers(ctx context.Context, prID string, need bool) error {
	if m.updateNeedMoreReviewers != nil {
		return m.updateNeedMoreReviewers(ctx, prID, need)
	}
	return nil
}

func (m *mockPullRequestRepo) GetPullRequest(ctx context.Context, prID string) (entities.PullRequest, error) {
	if m.getPullRequest != nil {
		return m.getPullRequest(ctx, prID)
	}
	return entities.PullRequest{}, nil
}

func TestUseCase_CreateTeam(t *testing.T) {
	teamRepo := &mockTeamRepo{createTeam: func(ctx context.Context, name string, members []entities.TeamMember) (entities.Team, error) {
		return entities.Team{Name: "backend", Members: []entities.TeamMember{{UserID: "ivan", Username: "Иван"}}}, nil
	}}
	uc := New(teamRepo, &mockPullRequestRepo{}, logger.New())
	result, err := uc.CreateTeam(context.Background(), entities.Team{Name: "backend"})
	assert.NoError(t, err)
	assert.Equal(t, "backend", result.Name)

	_, err = uc.CreateTeam(context.Background(), entities.Team{Name: ""})
	assert.Error(t, err)
}

func TestUseCase_GetTeam(t *testing.T) {
	teamRepo := &mockTeamRepo{getTeam: func(ctx context.Context, name string) (entities.Team, error) {
		return entities.Team{Name: "backend"}, nil
	}}
	uc := New(teamRepo, &mockPullRequestRepo{}, logger.New())
	result, err := uc.GetTeam(context.Background(), "backend")
	assert.NoError(t, err)
	assert.Equal(t, "backend", result.Name)

	_, err = uc.GetTeam(context.Background(), "")
	assert.Error(t, err)
}

func TestUseCase_SetUserActive(t *testing.T) {
	teamRepo := &mockTeamRepo{setUserActive: func(ctx context.Context, userID string, isActive bool) (entities.User, error) {
		return entities.User{ID: "ivan", IsActive: true}, nil
	}}
	uc := New(teamRepo, &mockPullRequestRepo{}, logger.New())
	result, err := uc.SetUserActive(context.Background(), "ivan", true)
	assert.NoError(t, err)
	assert.True(t, result.IsActive)

	_, err = uc.SetUserActive(context.Background(), "", true)
	assert.Error(t, err)
}

func TestUseCase_DeactivateTeamUsers(t *testing.T) {
	teamRepo := &mockTeamRepo{
		bulkSetUsersActive: func(ctx context.Context, teamName string, userIDs []string, isActive bool) ([]entities.User, error) {
			return []entities.User{{ID: "andrey", TeamName: "backend"}}, nil
		},
		listUsersByTeam: func(ctx context.Context, teamName string, onlyActive bool) ([]entities.User, error) {
			return []entities.User{{ID: "ivan"}}, nil
		},
	}
	prRepo := &mockPullRequestRepo{
		listOpenPullRequestsByReviewers: func(ctx context.Context, userIDs []string) (map[string][]entities.PullRequest, error) {
			return map[string][]entities.PullRequest{"andrey": {{ID: "pr-1", Status: entities.StatusOpen, AssignedReviewers: []string{"andrey"}, AuthorID: "ivan"}}}, nil
		},
		listAssignedReviewers:   func(ctx context.Context, prID string) ([]string, error) { return []string{"ivan"}, nil },
		replaceReviewer:         func(ctx context.Context, prID string, oldUserID string, newUserID *string) error { return nil },
		updateNeedMoreReviewers: func(ctx context.Context, prID string, need bool) error { return nil },
		getPullRequest: func(ctx context.Context, prID string) (entities.PullRequest, error) {
			return entities.PullRequest{ID: "pr-1", Status: entities.StatusOpen}, nil
		},
	}
	uc := New(teamRepo, prRepo, logger.New())
	result, err := uc.DeactivateTeamUsers(context.Background(), "backend", []string{"andrey"})
	assert.NoError(t, err)
	assert.Len(t, result.Users, 1)

	_, err = uc.DeactivateTeamUsers(context.Background(), "", []string{"andrey"})
	assert.Error(t, err)
}

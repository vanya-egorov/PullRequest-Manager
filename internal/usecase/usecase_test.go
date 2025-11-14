package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vanya-egorov/PullRequest-Manager/internal/entities"
	"github.com/vanya-egorov/PullRequest-Manager/pkg/logger"
)

type mockRepo struct {
	createTeam                      func(ctx context.Context, name string, members []entities.TeamMember) (entities.Team, error)
	getTeam                         func(ctx context.Context, name string) (entities.Team, error)
	getUser                         func(ctx context.Context, userID string) (entities.User, error)
	setUserActive                   func(ctx context.Context, userID string, isActive bool) (entities.User, error)
	listUsersByTeam                 func(ctx context.Context, teamName string, onlyActive bool) ([]entities.User, error)
	createPullRequest               func(ctx context.Context, pr entities.PullRequest) (entities.PullRequest, error)
	getPullRequest                  func(ctx context.Context, prID string) (entities.PullRequest, error)
	setPullRequestStatusMerged      func(ctx context.Context, prID string) (entities.PullRequest, error)
	listAssignedReviewers           func(ctx context.Context, prID string) ([]string, error)
	replaceReviewer                 func(ctx context.Context, prID string, oldUserID string, newUserID *string) error
	listReviewPullRequests          func(ctx context.Context, userID string) ([]entities.PullRequestShort, error)
	listReviewerAssignments         func(ctx context.Context) (map[string]int, error)
	countOpenPullRequests           func(ctx context.Context) (int, error)
	updateNeedMoreReviewers         func(ctx context.Context, prID string, need bool) error
	listOpenPullRequestsByReviewers func(ctx context.Context, userIDs []string) (map[string][]entities.PullRequest, error)
	bulkSetUsersActive              func(ctx context.Context, teamName string, userIDs []string, isActive bool) ([]entities.User, error)
}

func (m *mockRepo) CreateTeam(ctx context.Context, name string, members []entities.TeamMember) (entities.Team, error) {
	if m.createTeam != nil {
		return m.createTeam(ctx, name, members)
	}
	return entities.Team{}, nil
}

func (m *mockRepo) GetTeam(ctx context.Context, name string) (entities.Team, error) {
	if m.getTeam != nil {
		return m.getTeam(ctx, name)
	}
	return entities.Team{}, nil
}

func (m *mockRepo) GetUser(ctx context.Context, userID string) (entities.User, error) {
	if m.getUser != nil {
		return m.getUser(ctx, userID)
	}
	return entities.User{}, nil
}

func (m *mockRepo) SetUserActive(ctx context.Context, userID string, isActive bool) (entities.User, error) {
	if m.setUserActive != nil {
		return m.setUserActive(ctx, userID, isActive)
	}
	return entities.User{}, nil
}

func (m *mockRepo) ListUsersByTeam(ctx context.Context, teamName string, onlyActive bool) ([]entities.User, error) {
	if m.listUsersByTeam != nil {
		return m.listUsersByTeam(ctx, teamName, onlyActive)
	}
	return []entities.User{}, nil
}

func (m *mockRepo) CreatePullRequest(ctx context.Context, pr entities.PullRequest) (entities.PullRequest, error) {
	if m.createPullRequest != nil {
		return m.createPullRequest(ctx, pr)
	}
	return pr, nil
}

func (m *mockRepo) GetPullRequest(ctx context.Context, prID string) (entities.PullRequest, error) {
	if m.getPullRequest != nil {
		return m.getPullRequest(ctx, prID)
	}
	return entities.PullRequest{}, nil
}

func (m *mockRepo) SetPullRequestStatusMerged(ctx context.Context, prID string) (entities.PullRequest, error) {
	if m.setPullRequestStatusMerged != nil {
		return m.setPullRequestStatusMerged(ctx, prID)
	}
	return entities.PullRequest{}, nil
}

func (m *mockRepo) ListAssignedReviewers(ctx context.Context, prID string) ([]string, error) {
	if m.listAssignedReviewers != nil {
		return m.listAssignedReviewers(ctx, prID)
	}
	return []string{}, nil
}

func (m *mockRepo) ReplaceReviewer(ctx context.Context, prID string, oldUserID string, newUserID *string) error {
	if m.replaceReviewer != nil {
		return m.replaceReviewer(ctx, prID, oldUserID, newUserID)
	}
	return nil
}

func (m *mockRepo) ListReviewPullRequests(ctx context.Context, userID string) ([]entities.PullRequestShort, error) {
	if m.listReviewPullRequests != nil {
		return m.listReviewPullRequests(ctx, userID)
	}
	return []entities.PullRequestShort{}, nil
}

func (m *mockRepo) ListReviewerAssignments(ctx context.Context) (map[string]int, error) {
	if m.listReviewerAssignments != nil {
		return m.listReviewerAssignments(ctx)
	}
	return map[string]int{}, nil
}

func (m *mockRepo) CountOpenPullRequests(ctx context.Context) (int, error) {
	if m.countOpenPullRequests != nil {
		return m.countOpenPullRequests(ctx)
	}
	return 0, nil
}

func (m *mockRepo) UpdateNeedMoreReviewers(ctx context.Context, prID string, need bool) error {
	if m.updateNeedMoreReviewers != nil {
		return m.updateNeedMoreReviewers(ctx, prID, need)
	}
	return nil
}

func (m *mockRepo) ListOpenPullRequestsByReviewers(ctx context.Context, userIDs []string) (map[string][]entities.PullRequest, error) {
	if m.listOpenPullRequestsByReviewers != nil {
		return m.listOpenPullRequestsByReviewers(ctx, userIDs)
	}
	return map[string][]entities.PullRequest{}, nil
}

func (m *mockRepo) BulkSetUsersActive(ctx context.Context, teamName string, userIDs []string, isActive bool) ([]entities.User, error) {
	if m.bulkSetUsersActive != nil {
		return m.bulkSetUsersActive(ctx, teamName, userIDs, isActive)
	}
	return []entities.User{}, nil
}

func TestUseCase_CreateTeam(t *testing.T) {
	uc := New(&mockRepo{createTeam: func(ctx context.Context, name string, members []entities.TeamMember) (entities.Team, error) {
		return entities.Team{Name: "backend", Members: []entities.TeamMember{{UserID: "ivan", Username: "Иван"}}}, nil
	}}, logger.New())
	result, err := uc.CreateTeam(context.Background(), entities.Team{Name: "backend"})
	assert.NoError(t, err)
	assert.Equal(t, "backend", result.Name)

	_, err = uc.CreateTeam(context.Background(), entities.Team{Name: ""})
	assert.Error(t, err)
}

func TestUseCase_GetTeam(t *testing.T) {
	uc := New(&mockRepo{getTeam: func(ctx context.Context, name string) (entities.Team, error) {
		return entities.Team{Name: "backend"}, nil
	}}, logger.New())
	result, err := uc.GetTeam(context.Background(), "backend")
	assert.NoError(t, err)
	assert.Equal(t, "backend", result.Name)

	_, err = uc.GetTeam(context.Background(), "")
	assert.Error(t, err)
}

func TestUseCase_SetUserActive(t *testing.T) {
	uc := New(&mockRepo{setUserActive: func(ctx context.Context, userID string, isActive bool) (entities.User, error) {
		return entities.User{ID: "ivan", IsActive: true}, nil
	}}, logger.New())
	result, err := uc.SetUserActive(context.Background(), "ivan", true)
	assert.NoError(t, err)
	assert.True(t, result.IsActive)

	_, err = uc.SetUserActive(context.Background(), "", true)
	assert.Error(t, err)
}

func TestUseCase_CreatePullRequest(t *testing.T) {
	uc := New(&mockRepo{
		getUser: func(ctx context.Context, userID string) (entities.User, error) {
			return entities.User{ID: "ivan", TeamName: "backend"}, nil
		},
		listUsersByTeam: func(ctx context.Context, teamName string, onlyActive bool) ([]entities.User, error) {
			return []entities.User{{ID: "andrey"}, {ID: "dmitry"}}, nil
		},
		createPullRequest: func(ctx context.Context, pr entities.PullRequest) (entities.PullRequest, error) {
			return pr, nil
		},
	}, logger.New())
	result, err := uc.CreatePullRequest(context.Background(), CreatePullRequestInput{ID: "pr-1", Name: "Feature", AuthorID: "ivan"})
	assert.NoError(t, err)
	assert.Equal(t, "pr-1", result.ID)
	assert.Len(t, result.AssignedReviewers, 2)

	uc = New(&mockRepo{getUser: func(ctx context.Context, userID string) (entities.User, error) {
		return entities.User{}, entities.ErrUserNotFound
	}}, logger.New())
	_, err = uc.CreatePullRequest(context.Background(), CreatePullRequestInput{ID: "pr-1", Name: "Feature", AuthorID: "unknown"})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entities.ErrAuthorNotFound))

	_, err = uc.CreatePullRequest(context.Background(), CreatePullRequestInput{})
	assert.Error(t, err)
}

func TestUseCase_MergePullRequest(t *testing.T) {
	uc := New(&mockRepo{setPullRequestStatusMerged: func(ctx context.Context, prID string) (entities.PullRequest, error) {
		return entities.PullRequest{ID: "pr-1", Status: entities.StatusMerged}, nil
	}}, logger.New())
	result, err := uc.MergePullRequest(context.Background(), "pr-1")
	assert.NoError(t, err)
	assert.Equal(t, entities.StatusMerged, result.Status)

	_, err = uc.MergePullRequest(context.Background(), "")
	assert.Error(t, err)
}

func TestUseCase_ReassignReviewer(t *testing.T) {
	uc := New(&mockRepo{
		getPullRequest: func(ctx context.Context, prID string) (entities.PullRequest, error) {
			if prID == "pr-1" {
				return entities.PullRequest{ID: "pr-1", Status: entities.StatusOpen, AssignedReviewers: []string{"andrey"}, AuthorID: "ivan"}, nil
			}
			return entities.PullRequest{ID: "pr-1", AssignedReviewers: []string{"dmitry"}}, nil
		},
		getUser: func(ctx context.Context, userID string) (entities.User, error) {
			return entities.User{ID: "andrey", TeamName: "backend"}, nil
		},
		listUsersByTeam: func(ctx context.Context, teamName string, onlyActive bool) ([]entities.User, error) {
			return []entities.User{{ID: "dmitry"}}, nil
		},
		replaceReviewer:         func(ctx context.Context, prID string, oldUserID string, newUserID *string) error { return nil },
		listAssignedReviewers:   func(ctx context.Context, prID string) ([]string, error) { return []string{"dmitry"}, nil },
		updateNeedMoreReviewers: func(ctx context.Context, prID string, need bool) error { return nil },
	}, logger.New())
	result, err := uc.ReassignReviewer(context.Background(), "pr-1", "andrey")
	assert.NoError(t, err)
	assert.NotEmpty(t, result.ReplacedBy)

	uc = New(&mockRepo{getPullRequest: func(ctx context.Context, prID string) (entities.PullRequest, error) {
		return entities.PullRequest{ID: "pr-1", Status: entities.StatusMerged}, nil
	}}, logger.New())
	_, err = uc.ReassignReviewer(context.Background(), "pr-1", "andrey")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entities.ErrPullRequestMerged))

	uc = New(&mockRepo{getPullRequest: func(ctx context.Context, prID string) (entities.PullRequest, error) {
		return entities.PullRequest{ID: "pr-1", Status: entities.StatusOpen, AssignedReviewers: []string{"dmitry"}}, nil
	}}, logger.New())
	_, err = uc.ReassignReviewer(context.Background(), "pr-1", "andrey")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entities.ErrReviewerNotAssigned))
}

func TestUseCase_GetUserReviews(t *testing.T) {
	uc := New(&mockRepo{
		getUser: func(ctx context.Context, userID string) (entities.User, error) { return entities.User{ID: "ivan"}, nil },
		listReviewPullRequests: func(ctx context.Context, userID string) ([]entities.PullRequestShort, error) {
			return []entities.PullRequestShort{{ID: "pr-1", Name: "Feature"}}, nil
		},
	}, logger.New())
	result, err := uc.GetUserReviews(context.Background(), "ivan")
	assert.NoError(t, err)
	assert.Len(t, result, 1)

	_, err = uc.GetUserReviews(context.Background(), "")
	assert.Error(t, err)
}

func TestUseCase_GetStats(t *testing.T) {
	uc := New(&mockRepo{
		listReviewerAssignments: func(ctx context.Context) (map[string]int, error) { return map[string]int{"ivan": 5}, nil },
		countOpenPullRequests:   func(ctx context.Context) (int, error) { return 10, nil },
	}, logger.New())
	result, err := uc.GetStats(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 10, result.OpenPRs)
	assert.Equal(t, 5, result.AssignmentsByUser["ivan"])
}

func TestUseCase_DeactivateTeamUsers(t *testing.T) {
	uc := New(&mockRepo{
		bulkSetUsersActive: func(ctx context.Context, teamName string, userIDs []string, isActive bool) ([]entities.User, error) {
			return []entities.User{{ID: "andrey", TeamName: "backend"}}, nil
		},
		listOpenPullRequestsByReviewers: func(ctx context.Context, userIDs []string) (map[string][]entities.PullRequest, error) {
			return map[string][]entities.PullRequest{"andrey": {{ID: "pr-1", Status: entities.StatusOpen, AssignedReviewers: []string{"andrey"}, AuthorID: "ivan"}}}, nil
		},
		listUsersByTeam: func(ctx context.Context, teamName string, onlyActive bool) ([]entities.User, error) {
			return []entities.User{{ID: "ivan"}}, nil
		},
		listAssignedReviewers:   func(ctx context.Context, prID string) ([]string, error) { return []string{"ivan"}, nil },
		replaceReviewer:         func(ctx context.Context, prID string, oldUserID string, newUserID *string) error { return nil },
		updateNeedMoreReviewers: func(ctx context.Context, prID string, need bool) error { return nil },
		getPullRequest: func(ctx context.Context, prID string) (entities.PullRequest, error) {
			return entities.PullRequest{ID: "pr-1", Status: entities.StatusOpen}, nil
		},
	}, logger.New())
	result, err := uc.DeactivateTeamUsers(context.Background(), "backend", []string{"andrey"})
	assert.NoError(t, err)
	assert.Len(t, result.Users, 1)

	_, err = uc.DeactivateTeamUsers(context.Background(), "", []string{"andrey"})
	assert.Error(t, err)
}

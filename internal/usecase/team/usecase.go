package team

import (
	"context"
	"errors"
	"fmt"

	"github.com/vanya-egorov/PullRequest-Manager/internal/entities"
	"github.com/vanya-egorov/PullRequest-Manager/internal/repository"
	"github.com/vanya-egorov/PullRequest-Manager/pkg/logger"
	"github.com/vanya-egorov/PullRequest-Manager/pkg/random"
)

type useCase struct {
	teamRepo        repository.TeamRepository
	pullRequestRepo repository.PullRequestRepository
	rand            *random.Safe
	logger          logger.Logger
}

func New(teamRepo repository.TeamRepository, pullRequestRepo repository.PullRequestRepository, log logger.Logger) TeamUseCase {
	return &useCase{
		teamRepo:        teamRepo,
		pullRequestRepo: pullRequestRepo,
		rand:            random.New(),
		logger:          log,
	}
}

func (u *useCase) CreateTeam(ctx context.Context, team entities.Team) (entities.Team, error) {
	if team.Name == "" {
		return entities.Team{}, fmt.Errorf("team name required")
	}
	u.logger.Info("creating team", "name", team.Name)
	return u.teamRepo.CreateTeam(ctx, team.Name, team.Members)
}

func (u *useCase) GetTeam(ctx context.Context, name string) (entities.Team, error) {
	if name == "" {
		return entities.Team{}, fmt.Errorf("team name required")
	}
	return u.teamRepo.GetTeam(ctx, name)
}

func (u *useCase) SetUserActive(ctx context.Context, userID string, isActive bool) (entities.User, error) {
	if userID == "" {
		return entities.User{}, fmt.Errorf("user id required")
	}
	u.logger.Info("setting user active", "user_id", userID, "is_active", isActive)
	return u.teamRepo.SetUserActive(ctx, userID, isActive)
}

func (u *useCase) DeactivateTeamUsers(ctx context.Context, teamName string, userIDs []string) (DeactivateResult, error) {
	if teamName == "" {
		return DeactivateResult{}, fmt.Errorf("team name required")
	}

	u.logger.Info("deactivating team users", "team", teamName, "count", len(userIDs))
	updated, err := u.teamRepo.BulkSetUsersActive(ctx, teamName, userIDs, false)
	if err != nil {
		return DeactivateResult{}, err
	}

	affected, err := u.handleDeactivatedReviewers(ctx, teamName, updated)
	if err != nil {
		return DeactivateResult{}, err
	}

	u.logger.Info("team users deactivated", "team", teamName, "affected_prs", len(affected))
	return DeactivateResult{
		Users:         updated,
		AffectedPulls: affected,
	}, nil
}

func (u *useCase) handleDeactivatedReviewers(ctx context.Context, teamName string, deactivated []entities.User) ([]entities.PullRequest, error) {
	deactivatedIDs := make([]string, len(deactivated))
	for i, u := range deactivated {
		deactivatedIDs[i] = u.ID
	}

	openPRs, err := u.pullRequestRepo.ListOpenPullRequestsByReviewers(ctx, deactivatedIDs)
	if err != nil {
		return nil, err
	}

	activeMembers, err := u.teamRepo.ListUsersByTeam(ctx, teamName, true)
	if err != nil {
		return nil, err
	}

	activeSet := make(map[string]entities.User)
	for _, m := range activeMembers {
		activeSet[m.ID] = m
	}

	affectedMap := make(map[string]struct{})

	for reviewerID, prs := range openPRs {
		for _, pr := range prs {
			affectedMap[pr.ID] = struct{}{}
			if err := u.replaceDeactivatedReviewer(ctx, pr, reviewerID, activeSet); err != nil {
				return nil, err
			}
		}
	}

	return u.collectAffectedPRs(ctx, affectedMap)
}

func (u *useCase) replaceDeactivatedReviewer(ctx context.Context, pr entities.PullRequest, reviewerID string, activeSet map[string]entities.User) error {
	assignments, err := u.pullRequestRepo.ListAssignedReviewers(ctx, pr.ID)
	if err != nil {
		return err
	}

	assignedSet := make(map[string]struct{})
	for _, a := range assignments {
		assignedSet[a] = struct{}{}
	}
	delete(assignedSet, reviewerID)

	candidatePool := u.buildCandidatePool(activeSet, pr.AuthorID, assignedSet)

	if len(candidatePool) == 0 {
		return u.handleNoReplacement(ctx, pr, reviewerID)
	}

	newID := candidatePool[u.rand.Intn(len(candidatePool))]
	return u.replaceWithNewReviewer(ctx, pr, reviewerID, newID)
}

func (u *useCase) buildCandidatePool(activeSet map[string]entities.User, authorID string, assignedSet map[string]struct{}) []string {
	var candidatePool []string
	for id := range activeSet {
		if id == authorID {
			continue
		}
		if _, assigned := assignedSet[id]; assigned {
			continue
		}
		candidatePool = append(candidatePool, id)
	}
	return candidatePool
}

func (u *useCase) handleNoReplacement(ctx context.Context, pr entities.PullRequest, reviewerID string) error {
	if err := u.pullRequestRepo.ReplaceReviewer(ctx, pr.ID, reviewerID, nil); err != nil && !errors.Is(err, entities.ErrReviewerNotAssigned) {
		return err
	}

	assignments, err := u.pullRequestRepo.ListAssignedReviewers(ctx, pr.ID)
	if err != nil {
		return err
	}

	needMore := len(assignments) < 2
	return u.pullRequestRepo.UpdateNeedMoreReviewers(ctx, pr.ID, needMore)
}

func (u *useCase) replaceWithNewReviewer(ctx context.Context, pr entities.PullRequest, oldID, newID string) error {
	if err := u.pullRequestRepo.ReplaceReviewer(ctx, pr.ID, oldID, &newID); err != nil {
		return err
	}

	assignments, err := u.pullRequestRepo.ListAssignedReviewers(ctx, pr.ID)
	if err != nil {
		return err
	}

	needMore := len(assignments) < 2
	return u.pullRequestRepo.UpdateNeedMoreReviewers(ctx, pr.ID, needMore)
}

func (u *useCase) collectAffectedPRs(ctx context.Context, affectedMap map[string]struct{}) ([]entities.PullRequest, error) {
	var affected []entities.PullRequest
	for prID := range affectedMap {
		p, err := u.pullRequestRepo.GetPullRequest(ctx, prID)
		if err != nil {
			return nil, err
		}
		affected = append(affected, p)
	}
	return affected, nil
}

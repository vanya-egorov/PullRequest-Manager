package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/vanya-egorov/PullRequest-Manager/internal/entities"
	"github.com/vanya-egorov/PullRequest-Manager/internal/repository"
	"github.com/vanya-egorov/PullRequest-Manager/pkg/logger"
	"github.com/vanya-egorov/PullRequest-Manager/pkg/random"
)

type UseCase interface {
	CreateTeam(ctx context.Context, team entities.Team) (entities.Team, error)
	GetTeam(ctx context.Context, name string) (entities.Team, error)
	SetUserActive(ctx context.Context, userID string, isActive bool) (entities.User, error)
	CreatePullRequest(ctx context.Context, input CreatePullRequestInput) (entities.PullRequest, error)
	MergePullRequest(ctx context.Context, prID string) (entities.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID string, oldUserID string) (ReassignResult, error)
	GetUserReviews(ctx context.Context, userID string) ([]entities.PullRequestShort, error)
	GetStats(ctx context.Context) (entities.Stats, error)
	DeactivateTeamUsers(ctx context.Context, teamName string, userIDs []string) (DeactivateResult, error)
}

type useCase struct {
	teamRepo        repository.TeamRepository
	pullRequestRepo repository.PullRequestRepository
	statsRepo       repository.StatsRepository
	rand            *random.Safe
	logger          logger.Logger
}

func New(repo repository.Repository, log logger.Logger) UseCase {
	return &useCase{
		teamRepo:        repo,
		pullRequestRepo: repo,
		statsRepo:       repo,
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

type CreatePullRequestInput struct {
	ID       string
	Name     string
	AuthorID string
}

func (u *useCase) CreatePullRequest(ctx context.Context, input CreatePullRequestInput) (entities.PullRequest, error) {
	if input.ID == "" || input.Name == "" || input.AuthorID == "" {
		return entities.PullRequest{}, fmt.Errorf("invalid input")
	}

	u.logger.Debug("creating pull request", "id", input.ID, "author", input.AuthorID)
	author, err := u.teamRepo.GetUser(ctx, input.AuthorID)
	if err != nil {
		if errors.Is(err, entities.ErrUserNotFound) {
			return entities.PullRequest{}, entities.ErrAuthorNotFound
		}
		return entities.PullRequest{}, err
	}

	members, err := u.teamRepo.ListUsersByTeam(ctx, author.TeamName, true)
	if err != nil {
		return entities.PullRequest{}, err
	}

	candidates := u.filterCandidates(members, author.ID)
	selected := u.pickRandom(candidates, 2)
	needMore := len(selected) < 2

	pr := entities.PullRequest{
		ID:                input.ID,
		Name:              input.Name,
		AuthorID:          input.AuthorID,
		Status:            entities.StatusOpen,
		AssignedReviewers: selected,
		NeedMoreReviewers: needMore,
	}

	created, err := u.pullRequestRepo.CreatePullRequest(ctx, pr)
	if err != nil {
		return entities.PullRequest{}, err
	}
	u.logger.Info("pull request created", "id", created.ID, "reviewers", len(created.AssignedReviewers))
	return created, nil
}

func (u *useCase) MergePullRequest(ctx context.Context, prID string) (entities.PullRequest, error) {
	if prID == "" {
		return entities.PullRequest{}, fmt.Errorf("pr id required")
	}
	u.logger.Info("merging pull request", "id", prID)
	return u.pullRequestRepo.SetPullRequestStatusMerged(ctx, prID)
}

type ReassignResult struct {
	PullRequest entities.PullRequest
	ReplacedBy  string
}

func (u *useCase) ReassignReviewer(ctx context.Context, prID string, oldUserID string) (ReassignResult, error) {
	if prID == "" || oldUserID == "" {
		return ReassignResult{}, fmt.Errorf("invalid input")
	}

	u.logger.Debug("reassigning reviewer", "pr_id", prID, "old_user", oldUserID)
	pr, err := u.pullRequestRepo.GetPullRequest(ctx, prID)
	if err != nil {
		return ReassignResult{}, err
	}

	if pr.Status == entities.StatusMerged {
		return ReassignResult{}, entities.ErrPullRequestMerged
	}

	if !u.isReviewerAssigned(pr, oldUserID) {
		return ReassignResult{}, entities.ErrReviewerNotAssigned
	}

	newReviewer, err := u.findReplacement(ctx, pr, oldUserID)
	if err != nil {
		return ReassignResult{}, err
	}

	if err := u.pullRequestRepo.ReplaceReviewer(ctx, pr.ID, oldUserID, &newReviewer); err != nil {
		return ReassignResult{}, err
	}

	updated, err := u.updateReviewerCount(ctx, pr.ID)
	if err != nil {
		return ReassignResult{}, err
	}

	u.logger.Info("reviewer reassigned", "pr_id", prID, "old_user", oldUserID, "new_user", newReviewer)
	return ReassignResult{
		PullRequest: updated,
		ReplacedBy:  newReviewer,
	}, nil
}

func (u *useCase) GetUserReviews(ctx context.Context, userID string) ([]entities.PullRequestShort, error) {
	if userID == "" {
		return nil, fmt.Errorf("user id required")
	}

	if _, err := u.teamRepo.GetUser(ctx, userID); err != nil {
		return nil, err
	}

	return u.pullRequestRepo.ListReviewPullRequests(ctx, userID)
}

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

type DeactivateResult struct {
	Users         []entities.User
	AffectedPulls []entities.PullRequest
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

func (u *useCase) filterCandidates(members []entities.User, authorID string) []string {
	var candidates []string
	for _, m := range members {
		if m.ID != authorID {
			candidates = append(candidates, m.ID)
		}
	}
	return candidates
}

func (u *useCase) pickRandom(values []string, limit int) []string {
	if limit <= 0 || len(values) == 0 {
		return []string{}
	}
	if len(values) <= limit {
		return append([]string{}, values...)
	}

	result := make([]string, 0, limit)
	pool := append([]string{}, values...)

	for i := 0; i < limit && len(pool) > 0; i++ {
		idx := u.rand.Intn(len(pool))
		result = append(result, pool[idx])
		pool = append(pool[:idx], pool[idx+1:]...)
	}

	return result
}

func (u *useCase) isReviewerAssigned(pr entities.PullRequest, userID string) bool {
	for _, id := range pr.AssignedReviewers {
		if id == userID {
			return true
		}
	}
	return false
}

func (u *useCase) findReplacement(ctx context.Context, pr entities.PullRequest, oldUserID string) (string, error) {
	reviewer, err := u.teamRepo.GetUser(ctx, oldUserID)
	if err != nil {
		return "", err
	}

	candidates, err := u.teamRepo.ListUsersByTeam(ctx, reviewer.TeamName, true)
	if err != nil {
		return "", err
	}

	assignedSet := make(map[string]struct{})
	for _, id := range pr.AssignedReviewers {
		assignedSet[id] = struct{}{}
	}

	var available []string
	for _, candidate := range candidates {
		if candidate.ID == oldUserID || candidate.ID == pr.AuthorID {
			continue
		}
		if _, assigned := assignedSet[candidate.ID]; assigned {
			continue
		}
		available = append(available, candidate.ID)
	}

	if len(available) == 0 {
		return "", entities.ErrNoCandidate
	}

	return available[u.rand.Intn(len(available))], nil
}

func (u *useCase) updateReviewerCount(ctx context.Context, prID string) (entities.PullRequest, error) {
	updated, err := u.pullRequestRepo.GetPullRequest(ctx, prID)
	if err != nil {
		return entities.PullRequest{}, err
	}

	needMore := len(updated.AssignedReviewers) < 2
	if err := u.pullRequestRepo.UpdateNeedMoreReviewers(ctx, prID, needMore); err != nil {
		return entities.PullRequest{}, err
	}

	return updated, nil
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

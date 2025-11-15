package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/vanya-egorov/PullRequest-Manager/internal/entities"
)

type CreatePullRequestInput struct {
	ID       string
	Name     string
	AuthorID string
}

type ReassignResult struct {
	PullRequest entities.PullRequest
	ReplacedBy  string
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

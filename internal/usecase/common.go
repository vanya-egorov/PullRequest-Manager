package usecase

import (
	"github.com/vanya-egorov/PullRequest-Manager/internal/repository"
	"github.com/vanya-egorov/PullRequest-Manager/pkg/logger"
	"github.com/vanya-egorov/PullRequest-Manager/pkg/random"
)

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

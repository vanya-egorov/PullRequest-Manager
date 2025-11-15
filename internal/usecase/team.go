package usecase

import (
	"context"

	"github.com/vanya-egorov/PullRequest-Manager/internal/entities"
)

type TeamUseCase interface {
	CreateTeam(ctx context.Context, team entities.Team) (entities.Team, error)
	GetTeam(ctx context.Context, name string) (entities.Team, error)
	SetUserActive(ctx context.Context, userID string, isActive bool) (entities.User, error)
	DeactivateTeamUsers(ctx context.Context, teamName string, userIDs []string) (DeactivateResult, error)
}

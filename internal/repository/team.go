package repository

import (
	"context"

	"github.com/vanya-egorov/PullRequest-Manager/internal/entities"
)

type TeamRepository interface {
	CreateTeam(ctx context.Context, name string, members []entities.TeamMember) (entities.Team, error)
	GetTeam(ctx context.Context, name string) (entities.Team, error)
	GetUser(ctx context.Context, userID string) (entities.User, error)
	SetUserActive(ctx context.Context, userID string, isActive bool) (entities.User, error)
	ListUsersByTeam(ctx context.Context, teamName string, onlyActive bool) ([]entities.User, error)
	BulkSetUsersActive(ctx context.Context, teamName string, userIDs []string, isActive bool) ([]entities.User, error)
}

package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vanya-egorov/PullRequest-Manager/internal/entities"
	"github.com/vanya-egorov/PullRequest-Manager/internal/repository"
	"github.com/vanya-egorov/PullRequest-Manager/pkg/logger"
)

type PostgresRepository struct {
	pool   *pgxpool.Pool
	logger logger.Logger
}

func NewPostgresRepository(pool *pgxpool.Pool, log logger.Logger) repository.Repository {
	return &PostgresRepository{pool: pool, logger: log}
}

func (r *PostgresRepository) CreateTeam(ctx context.Context, name string, members []entities.TeamMember) (entities.Team, error) {
	r.logger.Debug("creating team", "name", name)
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		r.logger.Error("failed to begin transaction", "error", err)
		return entities.Team{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var teamID int64
	err = tx.QueryRow(ctx, "INSERT INTO teams (name) VALUES ($1) RETURNING id", name).Scan(&teamID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			r.logger.Error("team already exists", "name", name)
			return entities.Team{}, entities.ErrTeamExists
		}
		r.logger.Error("failed to insert team", "error", err)
		return entities.Team{}, err
	}

	for _, m := range members {
		_, err = tx.Exec(ctx, `INSERT INTO users (id, username, team_id, is_active) VALUES ($1,$2,$3,$4)
            ON CONFLICT (id) DO UPDATE SET username=EXCLUDED.username, team_id=EXCLUDED.team_id, is_active=EXCLUDED.is_active, updated_at=now()`,
			m.UserID, m.Username, teamID, m.IsActive,
		)
		if err != nil {
			r.logger.Error("failed to insert user", "user_id", m.UserID, "error", err)
			return entities.Team{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		r.logger.Error("failed to commit transaction", "error", err)
		return entities.Team{}, err
	}
	r.logger.Info("team created", "name", name)
	return r.GetTeam(ctx, name)
}

func (r *PostgresRepository) GetTeam(ctx context.Context, name string) (entities.Team, error) {
	var teamID int64
	err := r.pool.QueryRow(ctx, "SELECT id FROM teams WHERE name=$1", name).Scan(&teamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.Team{}, entities.ErrTeamNotFound
	}
	if err != nil {
		return entities.Team{}, err
	}

	rows, err := r.pool.Query(ctx, "SELECT id, username, is_active FROM users WHERE team_id=$1 ORDER BY username", teamID)
	if err != nil {
		return entities.Team{}, err
	}
	defer rows.Close()

	var members []entities.TeamMember
	for rows.Next() {
		var m entities.TeamMember
		if err = rows.Scan(&m.UserID, &m.Username, &m.IsActive); err != nil {
			return entities.Team{}, err
		}
		members = append(members, m)
	}

	return entities.Team{Name: name, Members: members}, nil
}

func (r *PostgresRepository) GetUser(ctx context.Context, userID string) (entities.User, error) {
	row := r.pool.QueryRow(ctx, `SELECT u.id, u.username, t.name, u.is_active FROM users u JOIN teams t ON t.id = u.team_id WHERE u.id=$1`, userID)
	var u entities.User
	err := row.Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.User{}, entities.ErrUserNotFound
	}
	if err != nil {
		return entities.User{}, err
	}
	return u, nil
}

func (r *PostgresRepository) SetUserActive(ctx context.Context, userID string, isActive bool) (entities.User, error) {
	row := r.pool.QueryRow(ctx, `UPDATE users SET is_active=$2, updated_at=now() WHERE id=$1 RETURNING id`, userID, isActive)
	var id string
	err := row.Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.User{}, entities.ErrUserNotFound
	}
	if err != nil {
		return entities.User{}, err
	}
	return r.GetUser(ctx, userID)
}

func (r *PostgresRepository) ListUsersByTeam(ctx context.Context, teamName string, onlyActive bool) ([]entities.User, error) {
	var rows pgx.Rows
	var err error
	if onlyActive {
		rows, err = r.pool.Query(ctx, `SELECT u.id, u.username, t.name, u.is_active FROM users u JOIN teams t ON t.id=u.team_id WHERE t.name=$1 AND u.is_active=true`, teamName)
	} else {
		rows, err = r.pool.Query(ctx, `SELECT u.id, u.username, t.name, u.is_active FROM users u JOIN teams t ON t.id=u.team_id WHERE t.name=$1`, teamName)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []entities.User
	for rows.Next() {
		var u entities.User
		if err = rows.Scan(&u.ID, &u.Username, &u.TeamName, &u.IsActive); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(users) == 0 {
		if _, err = r.GetTeam(ctx, teamName); err != nil {
			return nil, err
		}
	}
	return users, nil
}

func (r *PostgresRepository) CreatePullRequest(ctx context.Context, pr entities.PullRequest) (entities.PullRequest, error) {
	r.logger.Debug("creating pull request", "id", pr.ID)
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return entities.PullRequest{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `INSERT INTO pull_requests (id, name, author_id, status, need_more_reviewers) VALUES ($1,$2,$3,$4,$5)`,
		pr.ID, pr.Name, pr.AuthorID, string(pr.Status), pr.NeedMoreReviewers,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			r.logger.Error("pull request already exists", "id", pr.ID)
			return entities.PullRequest{}, entities.ErrPullRequestExists
		}
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			r.logger.Error("author not found", "author_id", pr.AuthorID)
			return entities.PullRequest{}, entities.ErrAuthorNotFound
		}
		return entities.PullRequest{}, err
	}

	for _, rev := range pr.AssignedReviewers {
		if _, err = tx.Exec(ctx, `INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES ($1,$2)`, pr.ID, rev); err != nil {
			return entities.PullRequest{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return entities.PullRequest{}, err
	}
	r.logger.Info("pull request created", "id", pr.ID)
	return r.GetPullRequest(ctx, pr.ID)
}

func (r *PostgresRepository) GetPullRequest(ctx context.Context, prID string) (entities.PullRequest, error) {
	row := r.pool.QueryRow(ctx, `SELECT id, name, author_id, status, need_more_reviewers, created_at, merged_at FROM pull_requests WHERE id=$1`, prID)
	var pr entities.PullRequest
	var status string
	var mergedAt *time.Time
	err := row.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &status, &pr.NeedMoreReviewers, &pr.CreatedAt, &mergedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return entities.PullRequest{}, entities.ErrPullRequestNotFound
	}
	if err != nil {
		return entities.PullRequest{}, err
	}
	pr.Status = entities.PullRequestStatus(status)
	pr.MergedAt = mergedAt

	reviewers, err := r.ListAssignedReviewers(ctx, prID)
	if err != nil {
		return entities.PullRequest{}, err
	}
	pr.AssignedReviewers = reviewers
	return pr, nil
}

func (r *PostgresRepository) SetPullRequestStatusMerged(ctx context.Context, prID string) (entities.PullRequest, error) {
	r.logger.Debug("merging pull request", "id", prID)
	_, err := r.pool.Exec(ctx, `UPDATE pull_requests SET status='MERGED', merged_at=COALESCE(merged_at, now()) WHERE id=$1`, prID)
	if err != nil {
		return entities.PullRequest{}, err
	}
	r.logger.Info("pull request merged", "id", prID)
	return r.GetPullRequest(ctx, prID)
}

func (r *PostgresRepository) ListAssignedReviewers(ctx context.Context, prID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT user_id FROM pull_request_reviewers WHERE pull_request_id=$1 ORDER BY assigned_at`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		reviewers = append(reviewers, id)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return reviewers, nil
}

func (r *PostgresRepository) ReplaceReviewer(ctx context.Context, prID string, oldUserID string, newUserID *string) error {
	r.logger.Debug("replacing reviewer", "pr_id", prID, "old_user", oldUserID)
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `DELETE FROM pull_request_reviewers WHERE pull_request_id=$1 AND user_id=$2`, prID, oldUserID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return entities.ErrReviewerNotAssigned
	}

	if newUserID != nil {
		if _, err = tx.Exec(ctx, `INSERT INTO pull_request_reviewers (pull_request_id, user_id) VALUES ($1,$2)`, prID, *newUserID); err != nil {
			return err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}
	r.logger.Info("reviewer replaced", "pr_id", prID, "old_user", oldUserID)
	return nil
}

func (r *PostgresRepository) ListReviewPullRequests(ctx context.Context, userID string) ([]entities.PullRequestShort, error) {
	rows, err := r.pool.Query(ctx, `SELECT p.id, p.name, p.author_id, p.status FROM pull_requests p JOIN pull_request_reviewers prr ON prr.pull_request_id=p.id WHERE prr.user_id=$1 ORDER BY p.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []entities.PullRequestShort
	for rows.Next() {
		var pr entities.PullRequestShort
		var status string
		if err = rows.Scan(&pr.ID, &pr.Name, &pr.AuthorID, &status); err != nil {
			return nil, err
		}
		pr.Status = entities.PullRequestStatus(status)
		prs = append(prs, pr)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return prs, nil
}

func (r *PostgresRepository) ListReviewerAssignments(ctx context.Context) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, `SELECT user_id, COUNT(*) FROM pull_request_reviewers GROUP BY user_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var id string
		var count int
		if err = rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		result[id] = count
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *PostgresRepository) CountOpenPullRequests(ctx context.Context) (int, error) {
	row := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM pull_requests WHERE status='OPEN'`)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *PostgresRepository) UpdateNeedMoreReviewers(ctx context.Context, prID string, need bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE pull_requests SET need_more_reviewers=$2 WHERE id=$1`, prID, need)
	return err
}

func (r *PostgresRepository) ListOpenPullRequestsByReviewers(ctx context.Context, userIDs []string) (map[string][]entities.PullRequest, error) {
	if len(userIDs) == 0 {
		return map[string][]entities.PullRequest{}, nil
	}
	rows, err := r.pool.Query(ctx, `SELECT prr.user_id, p.id, p.name, p.author_id, p.status, p.need_more_reviewers, p.created_at FROM pull_request_reviewers prr JOIN pull_requests p ON p.id=prr.pull_request_id WHERE prr.user_id = ANY($1::text[]) AND p.status='OPEN'`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]entities.PullRequest)
	for rows.Next() {
		var reviewer string
		var pr entities.PullRequest
		var status string
		if err = rows.Scan(&reviewer, &pr.ID, &pr.Name, &pr.AuthorID, &status, &pr.NeedMoreReviewers, &pr.CreatedAt); err != nil {
			return nil, err
		}
		pr.Status = entities.PullRequestStatus(status)
		result[reviewer] = append(result[reviewer], pr)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *PostgresRepository) BulkSetUsersActive(ctx context.Context, teamName string, userIDs []string, isActive bool) ([]entities.User, error) {
	r.logger.Debug("bulk setting users active", "team", teamName, "count", len(userIDs))
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var teamID int64
	err = tx.QueryRow(ctx, `SELECT id FROM teams WHERE name=$1`, teamName).Scan(&teamID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, entities.ErrTeamNotFound
	}
	if err != nil {
		return nil, err
	}

	var rows pgx.Rows
	if len(userIDs) == 0 {
		rows, err = tx.Query(ctx, `UPDATE users SET is_active=$2, updated_at=now() WHERE team_id=$1 RETURNING id, username, is_active`, teamID, isActive)
	} else {
		rows, err = tx.Query(ctx, `UPDATE users SET is_active=$3, updated_at=now() WHERE team_id=$1 AND id = ANY($2::text[]) RETURNING id, username, is_active`, teamID, userIDs, isActive)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []entities.User
	for rows.Next() {
		var u entities.User
		if err = rows.Scan(&u.ID, &u.Username, &u.IsActive); err != nil {
			return nil, err
		}
		u.TeamName = teamName
		result = append(result, u)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	if len(userIDs) > 0 {
		seen := make(map[string]struct{}, len(result))
		for _, u := range result {
			seen[u.ID] = struct{}{}
		}
		for _, id := range userIDs {
			if _, ok := seen[id]; !ok {
				return nil, entities.ErrUserNotFound
			}
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	r.logger.Info("users updated", "team", teamName, "count", len(result))
	return result, nil
}

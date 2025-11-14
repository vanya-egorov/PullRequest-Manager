package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/vanya-egorov/PullRequest-Manager/internal/handler"
	"github.com/vanya-egorov/PullRequest-Manager/internal/infrastructure/postgres"
	"github.com/vanya-egorov/PullRequest-Manager/internal/usecase"
	"github.com/vanya-egorov/PullRequest-Manager/pkg/logger"
)

func TestHTTPFlow(t *testing.T) {
	requireDocker(t)
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "pr_review",
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections"),
			wait.ForListeningPort("5432/tcp"),
		).WithDeadline(60 * time.Second),
	}

	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate container: %v", err)
		}
	})

	host, err := pgContainer.Host(ctx)
	require.NoError(t, err)

	port, err := pgContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf("postgres://postgres:postgres@%s:%s/pr_review?sslmode=disable", host, port.Port())

	require.Eventually(t, func() bool {
		pool, err := postgres.NewPool(ctx, dsn)
		if err != nil {
			return false
		}
		pool.Close()
		return true
	}, time.Minute, time.Second)

	pool, err := postgres.NewPool(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		pool.Close()
	})

	migrationsPath := getMigrationsPath(t)
	require.NoError(t, postgres.RunMigrations(ctx, pool, migrationsPath))

	log := logger.New()
	repo := postgres.NewPostgresRepository(pool, log)
	uc := usecase.New(repo, log)
	adminToken := "admin-secret"
	userToken := "user-secret"
	server := handler.New(uc, adminToken, userToken, log)
	ts := httptest.NewServer(server.Router())
	t.Cleanup(func() {
		ts.Close()
	})
	client := &http.Client{Timeout: 15 * time.Second}

	t.Run("team creation", func(t *testing.T) {
		body := map[string]any{
			"team_name": "backend",
			"members": []map[string]any{
				{"user_id": "u1", "username": "Ivan", "is_active": true},
				{"user_id": "u2", "username": "Vlad", "is_active": true},
				{"user_id": "u3", "username": "Andrey", "is_active": true},
				{"user_id": "u4", "username": "Dmitry", "is_active": true},
			},
		}
		resp := doRequest(t, client, ts.URL+"/team/add", http.MethodPost, body, "")
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		defer func() { _ = resp.Body.Close() }()
		var payload struct {
			Team struct {
				Members []struct {
					UserID string `json:"user_id"`
				} `json:"members"`
			} `json:"team"`
		}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
		require.Len(t, payload.Team.Members, 4)
	})

	createPR := func(id string) prResponse {
		body := map[string]any{
			"pull_request_id":   id,
			"pull_request_name": "Feature " + id,
			"author_id":         "u1",
		}
		resp := doRequest(t, client, ts.URL+"/pullRequest/create", http.MethodPost, body, adminToken)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		defer func() { _ = resp.Body.Close() }()
		var payload prResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
		return payload
	}

	pr1 := createPR("pr-1")
	require.Equal(t, "OPEN", pr1.PR.Status)
	require.True(t, len(pr1.PR.AssignedReviewers) > 0)

	pr2 := createPR("pr-2")
	require.Equal(t, "OPEN", pr2.PR.Status)

	reassignBody := map[string]any{
		"pull_request_id": pr1.PR.ID,
		"old_user_id":     pr1.PR.AssignedReviewers[0],
	}
	resp := doRequest(t, client, ts.URL+"/pullRequest/reassign", http.MethodPost, reassignBody, adminToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer func() { _ = resp.Body.Close() }()
	var reassignPayload struct {
		PR struct {
			AssignedReviewers []string `json:"assigned_reviewers"`
			Status            string   `json:"status"`
		} `json:"pr"`
		ReplacedBy string `json:"replaced_by"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&reassignPayload))
	require.Equal(t, "OPEN", reassignPayload.PR.Status)
	require.NotEmpty(t, reassignPayload.ReplacedBy)

	deactivateBody := map[string]any{
		"team_name": "backend",
		"user_ids":  []string{"u2", "u3"},
	}
	resp = doRequest(t, client, ts.URL+"/team/deactivate", http.MethodPost, deactivateBody, adminToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer func() { _ = resp.Body.Close() }()
	var deactivatePayload struct {
		PullRequests []struct {
			ID                string   `json:"pull_request_id"`
			AssignedReviewers []string `json:"assigned_reviewers"`
			NeedMore          bool     `json:"needMoreReviewers"`
		} `json:"pull_requests"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&deactivatePayload))
	for _, pr := range deactivatePayload.PullRequests {
		for _, reviewer := range pr.AssignedReviewers {
			require.NotContains(t, []string{"u2", "u3"}, reviewer)
		}
	}

	resp = doRequest(t, client, ts.URL+"/pullRequest/merge", http.MethodPost, map[string]any{"pull_request_id": pr1.PR.ID}, adminToken)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	_ = resp.Body.Close()

	resp = doRequest(t, client, ts.URL+"/pullRequest/reassign", http.MethodPost, reassignBody, adminToken)
	require.Equal(t, http.StatusConflict, resp.StatusCode)
	defer func() { _ = resp.Body.Close() }()
	var errorPayload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&errorPayload))
	require.Equal(t, "PR_MERGED", errorPayload.Error.Code)
}

func getMigrationsPath(t *testing.T) string {
	possiblePaths := []string{
		"db/migrations/postgresql",
		"../../db/migrations/postgresql",
		"../db/migrations/postgresql",
		"./db/migrations/postgresql",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	t.Fatalf("Migrations directory not found. Checked paths: %v", possiblePaths)
	return ""
}

type prResponse struct {
	PR struct {
		ID                string   `json:"pull_request_id"`
		Status            string   `json:"status"`
		AssignedReviewers []string `json:"assigned_reviewers"`
	} `json:"pr"`
}

func doRequest(t *testing.T, client *http.Client, url string, method string, body any, token string) *http.Response {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, reader)
	require.NoError(t, err)
	if method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	}
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

func requireDocker(t *testing.T) {
	paths := []string{
		"/var/run/docker.sock",
		filepath.Join(os.Getenv("HOME"), ".docker/run/docker.sock"),
	}
	if host := os.Getenv("DOCKER_HOST"); strings.HasPrefix(host, "unix://") {
		paths = append(paths, strings.TrimPrefix(host, "unix://"))
	}
	for _, p := range paths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			conn, dialErr := net.DialTimeout("unix", p, time.Second)
			if dialErr == nil {
				_ = conn.Close()
				return
			}
		}
	}
	t.Skip("docker socket not available")
}

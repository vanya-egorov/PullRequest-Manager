package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vanya-egorov/PullRequest-Manager/internal/entities"
	"github.com/vanya-egorov/PullRequest-Manager/internal/usecase"
	"github.com/vanya-egorov/PullRequest-Manager/pkg/logger"
)

type Handler struct {
	uc         usecase.UseCase
	adminToken string
	userToken  string
	logger     logger.Logger
}

func New(uc usecase.UseCase, adminToken, userToken string, log logger.Logger) *Handler {
	return &Handler{
		uc:         uc,
		adminToken: adminToken,
		userToken:  userToken,
		logger:     log,
	}
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()

	r.Post("/team/add", h.handleTeamAdd)
	r.Group(func(r chi.Router) {
		r.Use(h.authMiddleware(true, true))
		r.Get("/team/get", h.handleTeamGet)
		r.Get("/users/getReview", h.handleUserReviews)
	})
	r.Group(func(r chi.Router) {
		r.Use(h.authMiddleware(true, false))
		r.Post("/users/setIsActive", h.handleSetIsActive)
		r.Post("/pullRequest/create", h.handlePRCreate)
		r.Post("/pullRequest/merge", h.handlePRMerge)
		r.Post("/pullRequest/reassign", h.handlePRReassign)
		r.Get("/stats", h.handleStats)
		r.Post("/team/deactivate", h.handleTeamDeactivate)
	})
	return r
}

func (h *Handler) authMiddleware(allowAdmin bool, allowUser bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if h.authorized(r, allowAdmin, allowUser) {
				next.ServeHTTP(w, r)
				return
			}
			h.logger.Error("unauthorized request", "path", r.URL.Path)
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		})
	}
}

func (h *Handler) authorized(r *http.Request, allowAdmin bool, allowUser bool) bool {
	header := r.Header.Get("Authorization")
	if len(header) < 7 {
		return false
	}
	if header[:7] != "Bearer " {
		return false
	}
	token := header[7:]
	if allowAdmin && token == h.adminToken {
		return true
	}
	if allowUser && token == h.userToken {
		return true
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

type teamRequest struct {
	TeamName string             `json:"team_name"`
	Members  []teamMemberSchema `json:"members"`
}

type teamMemberSchema struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type teamResponse struct {
	Team teamSchema `json:"team"`
}

type teamSchema struct {
	TeamName string             `json:"team_name"`
	Members  []teamMemberSchema `json:"members"`
}

func toTeamSchema(team entities.Team) teamSchema {
	members := make([]teamMemberSchema, 0, len(team.Members))
	for _, m := range team.Members {
		members = append(members, teamMemberSchema{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}
	return teamSchema{
		TeamName: team.Name,
		Members:  members,
	}
}

func (h *Handler) handleTeamAdd(w http.ResponseWriter, r *http.Request) {
	var req teamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode team request", "error", err)
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid body")
		return
	}
	if req.TeamName == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name required")
		return
	}
	members := make([]entities.TeamMember, 0, len(req.Members))
	for _, m := range req.Members {
		if m.UserID == "" || m.Username == "" {
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid member")
			return
		}
		members = append(members, entities.TeamMember{
			UserID:   m.UserID,
			Username: m.Username,
			IsActive: m.IsActive,
		})
	}
	team, err := h.uc.CreateTeam(r.Context(), entities.Team{Name: req.TeamName, Members: members})
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, teamResponse{Team: toTeamSchema(team)})
}

func (h *Handler) handleTeamGet(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name required")
		return
	}
	team, err := h.uc.GetTeam(r.Context(), teamName)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toTeamSchema(team))
}

type setActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type userResponse struct {
	User userSchema `json:"user"`
}

type userSchema struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

func toUserSchema(u entities.User) userSchema {
	return userSchema{
		UserID:   u.ID,
		Username: u.Username,
		TeamName: u.TeamName,
		IsActive: u.IsActive,
	}
}

func (h *Handler) handleSetIsActive(w http.ResponseWriter, r *http.Request) {
	var req setActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode set active request", "error", err)
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid body")
		return
	}
	if req.UserID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "user_id required")
		return
	}
	user, err := h.uc.SetUserActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, userResponse{User: toUserSchema(user)})
}

type prCreateRequest struct {
	ID     string `json:"pull_request_id"`
	Name   string `json:"pull_request_name"`
	Author string `json:"author_id"`
}

type prResponse struct {
	PR prSchema `json:"pr"`
}

type prSchema struct {
	ID                string   `json:"pull_request_id"`
	Name              string   `json:"pull_request_name"`
	AuthorID          string   `json:"author_id"`
	Status            string   `json:"status"`
	AssignedReviewers []string `json:"assigned_reviewers"`
	NeedMoreReviewers bool     `json:"needMoreReviewers"`
	MergedAt          *string  `json:"mergedAt,omitempty"`
	CreatedAt         string   `json:"createdAt"`
}

func toPRSchema(pr entities.PullRequest) prSchema {
	var merged *string
	if pr.MergedAt != nil {
		formatted := pr.MergedAt.UTC().Format(time.RFC3339)
		merged = &formatted
	}
	return prSchema{
		ID:                pr.ID,
		Name:              pr.Name,
		AuthorID:          pr.AuthorID,
		Status:            string(pr.Status),
		AssignedReviewers: append([]string{}, pr.AssignedReviewers...),
		NeedMoreReviewers: pr.NeedMoreReviewers,
		CreatedAt:         pr.CreatedAt.UTC().Format(time.RFC3339),
		MergedAt:          merged,
	}
}

func (h *Handler) handlePRCreate(w http.ResponseWriter, r *http.Request) {
	var req prCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode PR create request", "error", err)
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid body")
		return
	}
	if req.ID == "" || req.Name == "" || req.Author == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "pull_request_id, pull_request_name and author_id required")
		return
	}
	pr, err := h.uc.CreatePullRequest(r.Context(), usecase.CreatePullRequestInput{
		ID:       req.ID,
		Name:     req.Name,
		AuthorID: req.Author,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, prResponse{PR: toPRSchema(pr)})
}

type prMergeRequest struct {
	ID string `json:"pull_request_id"`
}

func (h *Handler) handlePRMerge(w http.ResponseWriter, r *http.Request) {
	var req prMergeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode PR merge request", "error", err)
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid body")
		return
	}
	if req.ID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "pull_request_id required")
		return
	}
	pr, err := h.uc.MergePullRequest(r.Context(), req.ID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, prResponse{PR: toPRSchema(pr)})
}

type prReassignRequest struct {
	PRID    string `json:"pull_request_id"`
	OldUser string `json:"old_user_id"`
}

type prReassignResponse struct {
	PR         prSchema `json:"pr"`
	ReplacedBy string   `json:"replaced_by"`
}

func (h *Handler) handlePRReassign(w http.ResponseWriter, r *http.Request) {
	var req prReassignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode PR reassign request", "error", err)
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid body")
		return
	}
	if req.PRID == "" || req.OldUser == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "pull_request_id and old_user_id required")
		return
	}
	res, err := h.uc.ReassignReviewer(r.Context(), req.PRID, req.OldUser)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, prReassignResponse{PR: toPRSchema(res.PullRequest), ReplacedBy: res.ReplacedBy})
}

type userReviewsResponse struct {
	UserID       string          `json:"user_id"`
	PullRequests []prShortSchema `json:"pull_requests"`
}

type prShortSchema struct {
	ID       string `json:"pull_request_id"`
	Name     string `json:"pull_request_name"`
	AuthorID string `json:"author_id"`
	Status   string `json:"status"`
}

func (h *Handler) handleUserReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "user_id required")
		return
	}
	prs, err := h.uc.GetUserReviews(r.Context(), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	result := make([]prShortSchema, 0, len(prs))
	for _, pr := range prs {
		result = append(result, prShortSchema{
			ID:       pr.ID,
			Name:     pr.Name,
			AuthorID: pr.AuthorID,
			Status:   string(pr.Status),
		})
	}
	writeJSON(w, http.StatusOK, userReviewsResponse{UserID: userID, PullRequests: result})
}

type statsResponse struct {
	Assignments map[string]int `json:"assignments_by_user"`
	OpenPRs     int            `json:"open_prs"`
}

func (h *Handler) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.uc.GetStats(r.Context())
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, statsResponse{
		Assignments: stats.AssignmentsByUser,
		OpenPRs:     stats.OpenPRs,
	})
}

type deactivateRequest struct {
	TeamName string   `json:"team_name"`
	UserIDs  []string `json:"user_ids"`
}

type deactivateResponse struct {
	Users   []userSchema `json:"users"`
	Pullers []prSchema   `json:"pull_requests"`
}

func (h *Handler) handleTeamDeactivate(w http.ResponseWriter, r *http.Request) {
	var req deactivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to decode deactivate request", "error", err)
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid body")
		return
	}
	if req.TeamName == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "team_name required")
		return
	}
	result, err := h.uc.DeactivateTeamUsers(r.Context(), req.TeamName, req.UserIDs)
	if err != nil {
		h.handleError(w, err)
		return
	}
	users := make([]userSchema, 0, len(result.Users))
	for _, u := range result.Users {
		users = append(users, toUserSchema(u))
	}
	prs := make([]prSchema, 0, len(result.AffectedPulls))
	for _, pr := range result.AffectedPulls {
		prs = append(prs, toPRSchema(pr))
	}
	writeJSON(w, http.StatusOK, deactivateResponse{
		Users:   users,
		Pullers: prs,
	})
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	h.logger.Error("handler error", "error", err)
	switch {
	case errors.Is(err, entities.ErrTeamExists):
		writeError(w, http.StatusBadRequest, "TEAM_EXISTS", "team already exists")
	case errors.Is(err, entities.ErrTeamNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "team not found")
	case errors.Is(err, entities.ErrUserNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
	case errors.Is(err, entities.ErrPullRequestExists):
		writeError(w, http.StatusConflict, "PR_EXISTS", "pull request exists")
	case errors.Is(err, entities.ErrPullRequestNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "pull request not found")
	case errors.Is(err, entities.ErrPullRequestMerged):
		writeError(w, http.StatusConflict, "PR_MERGED", "pull request merged")
	case errors.Is(err, entities.ErrReviewerNotAssigned):
		writeError(w, http.StatusConflict, "NOT_ASSIGNED", "reviewer not assigned")
	case errors.Is(err, entities.ErrNoCandidate):
		writeError(w, http.StatusConflict, "NO_CANDIDATE", "no candidate available")
	case errors.Is(err, entities.ErrAuthorNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "author not found")
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
	}
}

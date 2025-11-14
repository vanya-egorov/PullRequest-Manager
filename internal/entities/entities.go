package entities

import "time"

type TeamMember struct {
	UserID   string
	Username string
	IsActive bool
}

type Team struct {
	Name    string
	Members []TeamMember
}

type User struct {
	ID       string
	Username string
	TeamName string
	IsActive bool
}

type PullRequestStatus string

const (
	StatusOpen   PullRequestStatus = "OPEN"
	StatusMerged PullRequestStatus = "MERGED"
)

type PullRequest struct {
	ID                string
	Name              string
	AuthorID          string
	Status            PullRequestStatus
	AssignedReviewers []string
	NeedMoreReviewers bool
	CreatedAt         time.Time
	MergedAt          *time.Time
}

type PullRequestShort struct {
	ID       string
	Name     string
	AuthorID string
	Status   PullRequestStatus
}

type Stats struct {
	AssignmentsByUser map[string]int
	OpenPRs           int
}

package entities

import "errors"

var (
	ErrTeamExists          = errors.New("team exists")
	ErrTeamNotFound        = errors.New("team not found")
	ErrUserNotFound        = errors.New("user not found")
	ErrAuthorNotFound      = errors.New("author not found")
	ErrPullRequestExists   = errors.New("pull request exists")
	ErrPullRequestNotFound = errors.New("pull request not found")
	ErrPullRequestMerged   = errors.New("pull request merged")
	ErrReviewerNotAssigned = errors.New("reviewer not assigned")
	ErrNoCandidate         = errors.New("no candidate available")
)

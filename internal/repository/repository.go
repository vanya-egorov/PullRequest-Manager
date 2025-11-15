package repository

type Repository interface {
	TeamRepository
	PullRequestRepository
	StatsRepository
}

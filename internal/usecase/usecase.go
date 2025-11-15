package usecase

type UseCase interface {
	TeamUseCase
	PullRequestUseCase
	StatsUseCase
}

package entities

type TeamMember struct {
	UserID   string
	Username string
	IsActive bool
}

type Team struct {
	Name    string
	Members []TeamMember
}

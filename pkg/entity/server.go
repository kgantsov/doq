package entity

type Server struct {
	Id         string
	Addr       string
	LeaderAddr string
	IsLeader   bool
	Suffrage   string
}

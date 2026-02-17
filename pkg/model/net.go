package model

type OpenPort struct {
	PID      int
	Port     int
	Address  string
	Protocol string
	State    string
}

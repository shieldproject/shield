package handler

type Topic string

const (
	Heartbeat = Topic("heartbeat")
	Alert     = Topic("alert")
	Shutdown  = Topic("shutdown")
)

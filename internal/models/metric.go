package models

import "time"

type Metric struct {
	ClientID         int32
	AppID            int32
	Requests         int32
	Landings         int32
	ApiRegisterTotal int32
	ApiRegisterScam  int32
	Timestamp        time.Time
}

type DailyFilter struct {
	ClientID int32
	AppID    *int32
	From     time.Time
	To       time.Time
}

type DailyMetric struct {
	Date             time.Time
	Requests         int64
	Landings         int64
	ApiRegisterTotal int64
	ApiRegisterScam  int64
}

package models

import "time"

type Metric struct {
	ID               int32
	ClientID         int32
	AppID            int32
	Requests         int32
	Landings         int32
	ApiRegisterTotal int32
	ApiRegisterScam  int32
	Timestamp        time.Time
}

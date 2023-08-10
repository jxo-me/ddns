package service

import "time"

type IDDNS interface {
	RunTimer(delay time.Duration)
	RunOnce()
}

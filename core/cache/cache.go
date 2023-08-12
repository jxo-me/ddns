package cache

type IIpCache interface {
	Check(string) bool
	IncreaseFailedTimes()
	ResetFailedTimes()
	GetFailedTimes() int
	GetTimes() int
	GetAddr() string
}

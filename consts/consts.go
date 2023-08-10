package consts

// UpdateStatusType 更新状态
type UpdateStatusType string

const (
	// UpdatedNothing 未改变
	UpdatedNothing UpdateStatusType = "UnChanged"
	// UpdatedFailed 更新失败
	UpdatedFailed = "Failure"
	// UpdatedSuccess 更新成功
	UpdatedSuccess = "Success"
)

const (
	HeaderAuthorization     = "Authorization"
	DefaultDDNSName         = "default"
	NetworkConnectedTimeout = 5
)

const (
	StatusReady   int32 = 0  // Job or Timer is ready for running.
	StatusRunning int32 = 1  // Job or Timer is already running.
	StatusStopped int32 = 2  // Job or Timer is stopped.
	StatusClosed  int32 = -1 // Job or Timer is closed and waiting to be deleted.
)

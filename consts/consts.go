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
	HeaderAuthorization = "Authorization"
)

package cache

import (
	"os"
	"strconv"
)

const IPCacheTimesENV = "DDNS_IP_CACHE_TIMES"

// IpCache 上次IP缓存
type IpCache struct {
	Addr          string // 缓存地址
	Times         int    // 剩余次数
	TimesFailedIP int    // 获取ip失败的次数
}

func (d *IpCache) Check(newAddr string) bool {
	if newAddr == "" {
		return true
	}
	// 地址改变 或 达到剩余次数
	if d.Addr != newAddr || d.Times <= 1 {
		IPCacheTimes, err := strconv.Atoi(os.Getenv(IPCacheTimesENV))
		if err != nil {
			IPCacheTimes = 5
		}
		d.Addr = newAddr
		d.Times = IPCacheTimes + 1
		return true
	}
	d.Addr = newAddr
	d.Times--
	return false
}

func (d *IpCache) IncreaseFailedTimes() {
	d.TimesFailedIP++
}

func (d *IpCache) ResetFailedTimes() {
	d.TimesFailedIP = 0
}

func (d *IpCache) GetFailedTimes() int {
	return d.TimesFailedIP
}

func (d *IpCache) GetTimes() int {
	return d.Times
}

func (d *IpCache) GetAddr() string {
	return d.Addr
}

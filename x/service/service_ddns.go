package service

import (
	"github.com/jxo-me/ddns/cache"
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/consts"
	"github.com/jxo-me/ddns/core/ddns"
	"github.com/jxo-me/ddns/core/logger"
	"github.com/jxo-me/ddns/internal/util"
	"github.com/jxo-me/ddns/x/hook"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

type DDNSService struct {
	DDNS               ddns.IDDNS
	IpCache            [2]cache.IpCache
	Conf               *config.DnsConfig
	Delay              time.Duration
	ForceCompareGlobal bool
	stop               chan chan struct{}
	status             *int32 // status is the current timer status.
	logger             logger.ILogger
}

func (s *DDNSService) String() string {
	return s.DDNS.String()
}

func NewDDNS(d ddns.IDDNS, log logger.ILogger) *DDNSService {
	st := consts.StatusRunning
	s := &DDNSService{
		DDNS:               d,
		stop:               make(chan chan struct{}),
		ForceCompareGlobal: true,
		status:             &st,
		logger:             log,
	}

	return s
}

func (s *DDNSService) RunOnce() {
	if s.ForceCompareGlobal {
		s.IpCache = [2]cache.IpCache{{}, {}}
	}
	s.DDNS.Init(s.Conf, &s.IpCache[0], &s.IpCache[1])
	domains := s.DDNS.AddUpdateDomainRecords()
	// webhook
	if s.Conf.Webhook != nil {
		webhook := hook.NewHook(s.Conf.Webhook.WebhookURL, s.Conf.Webhook.WebhookRequestBody, s.Conf.Webhook.WebhookHeaders)
		v4Status, v6Status := webhook.ExecHook(&domains)
		// 重置单个cache
		if v4Status == consts.UpdatedFailed {
			s.IpCache[0] = cache.IpCache{}
		}
		if v6Status == consts.UpdatedFailed {
			s.IpCache[1] = cache.IpCache{}
		}
	}

	s.ForceCompareGlobal = false
}

func (s *DDNSService) Loop(confirm chan struct{}) {
	go func(stop chan struct{}) {
		var (
			timerIntervalTicker = time.NewTicker(s.Delay)
		)
		defer timerIntervalTicker.Stop()
		for {
			select {
			case <-timerIntervalTicker.C:
				// Check the timer status.
				switch atomic.LoadInt32(s.status) {
				case consts.StatusRunning:
					// Timer proceeding.
					st := consts.StatusRunning
					s.status = &st
					s.RunOnce()
					close(stop)
				case consts.StatusStopped:
					// Do nothing.
				case consts.StatusClosed:
					// Timer exits.
					return
				}
			}
		}
	}(confirm)
}

func (s *DDNSService) Start() error {
	// 等待网络连接
	s.waitForNetworkConnected()

	stop := make(chan struct{})
	stopConfirm := make(chan struct{})
	s.Loop(stopConfirm)

	for {
		select {
		// call to stop polling
		case confirm := <-s.stop:
			close(stop)
			<-stopConfirm
			close(confirm)
			return nil
		}
	}
}

func (s *DDNSService) Stop() error {
	st := consts.StatusClosed
	s.status = &st
	confirm := make(chan struct{})
	s.stop <- confirm
	<-confirm

	return nil
}

// waitForNetworkConnected 等待网络连接后继续
func (s *DDNSService) waitForNetworkConnected() {
	// 延时 5 秒
	timeout := time.Second * consts.NetworkConnectedTimeout
	// 等待网络连接
	loopbackServer := "[::1]:53"
	find := false
	addr := s.DDNS.Endpoint()
	if addr != "" {
		for {
			client := util.CreateHTTPClient()
			resp, err := client.Get(addr)
			if err != nil {
				// 如果 err 包含回环地址（[::1]:53）则表示没有 DNS 服务器，设置 DNS 服务器
				if strings.Contains(err.Error(), loopbackServer) && !find {
					server := "1.1.1.1:53"
					s.logger.Debugf("解析回环地址 %s 失败！将默认使用 %s，可参考文档通过 -dns 自定义 DNS 服务器",
						loopbackServer, server)

					_ = os.Setenv(util.DNSServerEnv, server)
					find = true
					continue
				}

				s.logger.Debugf("等待网络连接：%s。%s 后重试...", err, timeout)
				// 等待 5 秒后重试
				time.Sleep(timeout)
				continue
			}

			// 网络已连接
			_ = resp.Body.Close()
			return
		}
	}
	find = true
}

package service

import (
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/consts"
	iCache "github.com/jxo-me/ddns/core/cache"
	"github.com/jxo-me/ddns/core/ddns"
	"github.com/jxo-me/ddns/core/logger"
	"github.com/jxo-me/ddns/internal/util"
	"github.com/jxo-me/ddns/sdk/cache"
	"github.com/jxo-me/ddns/sdk/hook"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

type DDNSService struct {
	DDNS               ddns.IDDNS
	IpCache            [2]iCache.IIpCache
	Conf               *config.DDnsConfig
	Delay              time.Duration
	ForceCompareGlobal bool
	stop               chan chan struct{}
	status             *int32 // status is the current timer status.
	logger             logger.ILogger
}

func (s *DDNSService) String() string {
	return s.DDNS.String()
}

func NewDDNS(d ddns.IDDNS, log logger.ILogger, conf *config.DDnsConfig) *DDNSService {
	st := consts.StatusRunning
	s := &DDNSService{
		DDNS:               d,
		stop:               make(chan chan struct{}),
		ForceCompareGlobal: true,
		status:             &st,
		logger:             log,
		Delay:              time.Second * time.Duration(conf.Delay),
		Conf:               conf,
	}

	return s
}

func (s *DDNSService) Run() {
	if s.ForceCompareGlobal {
		s.IpCache = [2]iCache.IIpCache{&cache.IpCache{}, &cache.IpCache{}}
	}
	s.DDNS.Init(s.Conf, s.IpCache[0], s.IpCache[1], s.logger)
	domains := s.DDNS.AddUpdateDomainRecords()
	// webhook
	if s.Conf.Webhook != nil {
		webhook := hook.NewHook(s.Conf.Webhook.WebhookURL, s.Conf.Webhook.WebhookRequestBody,
			s.Conf.Webhook.WebhookHeaders, s.logger)
		v4Status, v6Status := webhook.ExecHook(&domains)
		// 重置单个cache
		if v4Status == consts.UpdatedFailed {
			s.IpCache[0] = &cache.IpCache{}
		}
		if v6Status == consts.UpdatedFailed {
			s.IpCache[1] = &cache.IpCache{}
		}
	}

	s.ForceCompareGlobal = false
}

func (s *DDNSService) Worker() error {
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
				s.logger.Debugf("%s DDNS service is running!", s.DDNS.String())
				// Timer proceeding.
				s.Run()
			case consts.StatusStopped:
				s.logger.Debugf("%s DDNS service has been stopped!", s.DDNS.String())
				// Do nothing.
			case consts.StatusClosed:
				// Timer exits.
				s.logger.Debugf("%s DDNS service is closed!", s.DDNS.String())
			}
		// call to stop polling
		case confirm := <-s.stop:
			close(confirm)
			s.logger.Debugf("%s DDNS service has been manually stopped!", s.DDNS.String())
			return nil
		}
	}
}

func (s *DDNSService) Start() error {
	// 等待网络连接
	s.waitForNetworkConnected()
	// 启动服务
	return s.Worker()
}

func (s *DDNSService) Stop() error {
	st := consts.StatusStopped
	s.status = &st
	confirm := make(chan struct{})
	s.stop <- confirm

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
					s.logger.Debugf("Failed to resolve loopback address %s! %s will be used by default, you can refer to the documentation to customize the DNS server through -dns", loopbackServer, server)

					_ = os.Setenv(util.DNSServerEnv, server)
					find = true
					continue
				}

				s.logger.Debugf("Waiting for network connection: %s. Try again in %s...", err, timeout)
				// 等待 5 秒后重试
				time.Sleep(timeout)
				continue
			}
			s.logger.Debugf("The network is connected: %s", addr)
			// 网络已连接
			_ = resp.Body.Close()
			return
		}
	}
	find = true
}

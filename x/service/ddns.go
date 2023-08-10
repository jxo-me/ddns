package service

import (
	"github.com/jxo-me/ddns/cache"
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/consts"
	"github.com/jxo-me/ddns/core/ddns"
	"github.com/jxo-me/ddns/core/service"
	"github.com/jxo-me/ddns/internal/util"
	"github.com/jxo-me/ddns/x/ddns/alidns"
	"github.com/jxo-me/ddns/x/ddns/baidu"
	"github.com/jxo-me/ddns/x/ddns/callback"
	"github.com/jxo-me/ddns/x/ddns/cloudflare"
	"github.com/jxo-me/ddns/x/ddns/dnspod"
	"github.com/jxo-me/ddns/x/ddns/godaddy"
	"github.com/jxo-me/ddns/x/ddns/google"
	"github.com/jxo-me/ddns/x/ddns/huawei"
	"github.com/jxo-me/ddns/x/ddns/namecheap"
	"github.com/jxo-me/ddns/x/ddns/namesilo"
	"github.com/jxo-me/ddns/x/ddns/porkbun"
	"github.com/jxo-me/ddns/x/ddns/tencent"
	"github.com/jxo-me/ddns/x/hook"
	"log"
	"os"
	"strings"
	"time"
)

var (
	IpCache            [][2]cache.IpCache
	ForceCompareGlobal = true
)

func NewDDNS() service.IDDNS {

	s := &dDnsService{}

	return s
}

type dDnsService struct{}

func (s *dDnsService) RunTimer(delay time.Duration) {
	waitForNetworkConnected()

	for {
		s.RunOnce()
		time.Sleep(delay)
	}
}

func (s *dDnsService) RunOnce() {
	conf, err := config.GetConfigCached()
	if err != nil {
		return
	}
	if ForceCompareGlobal || len(IpCache) != len(conf.DnsConf) {
		IpCache = [][2]cache.IpCache{}
		for range conf.DnsConf {
			IpCache = append(IpCache, [2]cache.IpCache{{}, {}})
		}
	}

	for i, dc := range conf.DnsConf {
		var dnsSelected ddns.IDDNS
		switch dc.DNS.Name {
		case alidns.Code:
			dnsSelected = &alidns.Alidns{}
		case tencent.Code:
			dnsSelected = &tencent.TencentCloud{}
		case dnspod.Code:
			dnsSelected = &dnspod.Dnspod{}
		case cloudflare.Code:
			dnsSelected = &cloudflare.Cloudflare{}
		case huawei.Code:
			dnsSelected = &huawei.Huaweicloud{}
		case callback.Code:
			dnsSelected = &callback.Callback{}
		case baidu.Code:
			dnsSelected = &baidu.BaiduCloud{}
		case porkbun.Code:
			dnsSelected = &porkbun.Porkbun{}
		case godaddy.Code:
			dnsSelected = &godaddy.GoDaddyDNS{}
		case google.Code:
			dnsSelected = &google.GoogleDomain{}
		case namecheap.Code:
			dnsSelected = &namecheap.NameCheap{}
		case namesilo.Code:
			dnsSelected = &namesilo.NameSilo{}
		default:
			dnsSelected = &alidns.Alidns{}
		}
		dnsSelected.Init(&dc, &IpCache[i][0], &IpCache[i][1])
		domains := dnsSelected.AddUpdateDomainRecords()
		// webhook
		webhook := hook.NewHook(conf.WebhookURL, conf.WebhookRequestBody, conf.WebhookHeaders)
		v4Status, v6Status := webhook.ExecHook(&domains)
		// 重置单个cache
		if v4Status == consts.UpdatedFailed {
			IpCache[i][0] = cache.IpCache{}
		}
		if v6Status == consts.UpdatedFailed {
			IpCache[i][1] = cache.IpCache{}
		}
	}

	ForceCompareGlobal = false

}

// waitForNetworkConnected 等待网络连接后继续
func waitForNetworkConnected() {
	// 延时 5 秒
	timeout := time.Second * 5

	// 测试网络是否连接的域名
	addresses := []string{
		alidns.Endpoint,
		baidu.Endpoint,
		cloudflare.Endpoint,
		dnspod.Endpoint,
		google.Endpoint,
		huawei.Endpoint,
		namecheap.Endpoint,
		namesilo.Endpoint,
		porkbun.Endpoint,
		tencent.EndPoint,
	}

	loopbackServer := "[::1]:53"
	find := false

	for {
		for _, addr := range addresses {
			// https://github.com/jeessy2/ddns-go/issues/736
			client := util.CreateHTTPClient()
			resp, err := client.Get(addr)
			if err != nil {

				// 如果 err 包含回环地址（[::1]:53）则表示没有 DNS 服务器，设置 DNS 服务器
				if strings.Contains(err.Error(), loopbackServer) && !find {
					server := "1.1.1.1:53"
					log.Printf("解析回环地址 %s 失败！将默认使用 %s，可参考文档通过 -dns 自定义 DNS 服务器",
						loopbackServer, server)

					_ = os.Setenv(util.DNSServerEnv, server)
					find = true
					continue
				}

				log.Printf("等待网络连接：%s。%s 后重试...", err, timeout)
				// 等待 5 秒后重试
				time.Sleep(timeout)
				continue
			}

			// 网络已连接
			_ = resp.Body.Close()
			return
		}
	}
}

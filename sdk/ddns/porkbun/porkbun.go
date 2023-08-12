package porkbun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jxo-me/ddns/config"
	"github.com/jxo-me/ddns/consts"
	"github.com/jxo-me/ddns/core/cache"
	"github.com/jxo-me/ddns/core/logger"
	"github.com/jxo-me/ddns/internal/util"
	"github.com/jxo-me/ddns/sdk/ddns"
	"net/http"
)

const (
	Endpoint string = "https://porkbun.com/api/json/v3/dns"
	Code     string = "porkbun"
)

type Porkbun struct {
	DNSConfig *config.DNS
	Domains   ddns.Domains
	TTL       string
	logger    logger.ILogger
}
type PorkbunDomainRecord struct {
	Name    *string `json:"name"`    // subdomain
	Type    *string `json:"type"`    // record type, e.g. A AAAA CNAME
	Content *string `json:"content"` // value
	Ttl     *string `json:"ttl"`     // default 300
}

type PorkbunResponse struct {
	Status string `json:"status"`
}

type PorkbunDomainQueryResponse struct {
	*PorkbunResponse
	Records []PorkbunDomainRecord `json:"records"`
}

type PorkbunApiKey struct {
	AccessKey string `json:"apikey"`
	SecretKey string `json:"secretapikey"`
}

type PorkbunDomainCreateOrUpdateVO struct {
	*PorkbunApiKey
	*PorkbunDomainRecord
}

func (pb *Porkbun) String() string {
	return Code
}

func (pb *Porkbun) Endpoint() string {
	return Endpoint
}

// Init 初始化
func (pb *Porkbun) Init(conf *config.DDnsConfig, ipv4cache cache.IIpCache, ipv6cache cache.IIpCache, log logger.ILogger) {
	pb.Domains.Ipv4Cache = ipv4cache
	pb.Domains.Ipv6Cache = ipv6cache
	pb.Domains.Logger = log
	pb.DNSConfig = conf.DNS
	pb.Domains.GetNewIp(conf)
	pb.logger = log
	if conf.TTL == "" {
		// 默认600s
		pb.TTL = "600"
	} else {
		pb.TTL = conf.TTL
	}
}

// AddUpdateDomainRecords 添加或更新IPv4/IPv6记录
func (pb *Porkbun) AddUpdateDomainRecords() ddns.Domains {
	pb.addUpdateDomainRecords("A")
	pb.addUpdateDomainRecords("AAAA")
	return pb.Domains
}

func (pb *Porkbun) addUpdateDomainRecords(recordType string) {
	ipAddr, domains := pb.Domains.GetNewIpResult(recordType)

	if ipAddr == "" {
		return
	}

	for _, domain := range domains {
		var record PorkbunDomainQueryResponse
		// 获取当前域名信息
		err := pb.request(
			Endpoint+fmt.Sprintf("/retrieveByNameType/%s/%s/%s", domain.DomainName, recordType, domain.SubDomain),
			&PorkbunApiKey{
				AccessKey: pb.DNSConfig.ID,
				SecretKey: pb.DNSConfig.Secret,
			},
			&record,
		)

		if err != nil {
			domain.UpdateStatus = consts.UpdatedFailed
			return
		}
		if record.Status == "SUCCESS" {
			if len(record.Records) > 0 {
				// 存在，更新
				pb.modify(&record, domain, &recordType, &ipAddr)
			} else {
				// 不存在，创建
				pb.create(domain, &recordType, &ipAddr)
			}
		} else {
			pb.logger.Infof("查询现有域名记录失败")
			domain.UpdateStatus = consts.UpdatedFailed
		}
	}
}

// 创建
func (pb *Porkbun) create(domain *ddns.Domain, recordType *string, ipAddr *string) {
	var response PorkbunResponse

	err := pb.request(
		Endpoint+fmt.Sprintf("/create/%s", domain.DomainName),
		&PorkbunDomainCreateOrUpdateVO{
			PorkbunApiKey: &PorkbunApiKey{
				AccessKey: pb.DNSConfig.ID,
				SecretKey: pb.DNSConfig.Secret,
			},
			PorkbunDomainRecord: &PorkbunDomainRecord{
				Name:    &domain.SubDomain,
				Type:    recordType,
				Content: ipAddr,
				Ttl:     &pb.TTL,
			},
		},
		&response,
	)

	if err == nil && response.Status == "SUCCESS" {
		pb.logger.Infof("新增域名解析 %s 成功！IP: %s", domain, *ipAddr)
		domain.UpdateStatus = consts.UpdatedSuccess
	} else {
		pb.logger.Infof("新增域名解析 %s 失败！", domain)
		domain.UpdateStatus = consts.UpdatedFailed
	}
}

// 修改
func (pb *Porkbun) modify(record *PorkbunDomainQueryResponse, domain *ddns.Domain, recordType *string, ipAddr *string) {

	// 相同不修改
	if len(record.Records) > 0 && *record.Records[0].Content == *ipAddr {
		pb.logger.Infof("你的IP %s 没有变化, 域名 %s", *ipAddr, domain)
		return
	}

	var response PorkbunResponse

	err := pb.request(
		Endpoint+fmt.Sprintf("/editByNameType/%s/%s/%s", domain.DomainName, *recordType, domain.SubDomain),
		&PorkbunDomainCreateOrUpdateVO{
			PorkbunApiKey: &PorkbunApiKey{
				AccessKey: pb.DNSConfig.ID,
				SecretKey: pb.DNSConfig.Secret,
			},
			PorkbunDomainRecord: &PorkbunDomainRecord{
				Content: ipAddr,
				Ttl:     &pb.TTL,
			},
		},
		&response,
	)

	if err == nil && response.Status == "SUCCESS" {
		pb.logger.Infof("更新域名解析 %s 成功！IP: %s", domain, *ipAddr)
		domain.UpdateStatus = consts.UpdatedSuccess
	} else {
		pb.logger.Infof("更新域名解析 %s 失败！", domain)
		domain.UpdateStatus = consts.UpdatedFailed
	}
}

// request 统一请求接口
func (pb *Porkbun) request(url string, data interface{}, result interface{}) (err error) {
	jsonStr := make([]byte, 0)
	if data != nil {
		jsonStr, _ = json.Marshal(data)
	}
	req, err := http.NewRequest(
		"POST",
		url,
		bytes.NewBuffer(jsonStr),
	)
	if err != nil {
		pb.logger.Infof("http.NewRequest失败. Error: ", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := util.CreateHTTPClient()
	resp, err := client.Do(req)
	err = util.GetHTTPResponse(resp, url, err, result)

	return
}

package config

import (
	"github.com/jxo-me/ddns/consts"
	"net/url"
)

// Domain 域名实体
type Domain struct {
	DomainName   string
	SubDomain    string
	CustomParams string
	UpdateStatus consts.UpdateStatusType // 更新状态
}

func (d Domain) String() string {
	if d.SubDomain != "" {
		return d.SubDomain + "." + d.DomainName
	}
	return d.DomainName
}

// GetFullDomain 获得全部的，子域名
func (d Domain) GetFullDomain() string {
	if d.SubDomain != "" {
		return d.SubDomain + "." + d.DomainName
	}
	return "@" + "." + d.DomainName
}

// GetSubDomain 获得子域名，为空返回@
// 阿里云/腾讯云/dnspod/GoDaddy/namecheap 需要
func (d Domain) GetSubDomain() string {
	if d.SubDomain != "" {
		return d.SubDomain
	}
	return "@"
}

// GetCustomParams not be nil
func (d Domain) GetCustomParams() url.Values {
	if d.CustomParams != "" {
		q, err := url.ParseQuery(d.CustomParams)
		if err == nil {
			return q
		}
	}
	return url.Values{}
}

package config

import (
	"github.com/jxo-me/ddns/core/logger"
	"github.com/jxo-me/ddns/internal/util"
	"io"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Ipv4Reg IPv4正则
var Ipv4Reg = regexp.MustCompile(`((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])`)

// Ipv6Reg IPv6正则
var Ipv6Reg = regexp.MustCompile(`((([0-9A-Fa-f]{1,4}:){7}([0-9A-Fa-f]{1,4}|:))|(([0-9A-Fa-f]{1,4}:){6}(:[0-9A-Fa-f]{1,4}|((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){5}(((:[0-9A-Fa-f]{1,4}){1,2})|:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3})|:))|(([0-9A-Fa-f]{1,4}:){4}(((:[0-9A-Fa-f]{1,4}){1,3})|((:[0-9A-Fa-f]{1,4})?:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){3}(((:[0-9A-Fa-f]{1,4}){1,4})|((:[0-9A-Fa-f]{1,4}){0,2}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){2}(((:[0-9A-Fa-f]{1,4}){1,5})|((:[0-9A-Fa-f]{1,4}){0,3}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(([0-9A-Fa-f]{1,4}:){1}(((:[0-9A-Fa-f]{1,4}){1,6})|((:[0-9A-Fa-f]{1,4}){0,4}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:))|(:(((:[0-9A-Fa-f]{1,4}){1,7})|((:[0-9A-Fa-f]{1,4}){0,5}:((25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}))|:)))`)

// DNS DNS配置
type DNS struct {
	// 名称。如：alidns,webhook
	Name   string `json:"name"`
	ID     string `json:"ID"`
	Secret string `json:"secret"`
}

type Ipv4 struct {
	Enable bool `json:"enable"`
	// 获取IP类型 url/netInterface
	GetType      string   `yaml:",omitempty" json:"getType"` // url/netInterface/cmd
	URL          string   `yaml:",omitempty" json:"url"`
	NetInterface string   `yaml:",omitempty" json:"netInterface"`
	Cmd          string   `yaml:",omitempty" json:"cmd"`
	Domains      []string `yaml:",omitempty" json:"domains"`
}

type Ipv6 struct {
	Enable bool `json:"enable"`
	// 获取IP类型 url/netInterface
	GetType      string `yaml:",omitempty" json:"getType"`
	URL          string `yaml:",omitempty" json:"url"`
	NetInterface string `yaml:",omitempty" json:"netInterface"`
	Cmd          string `yaml:",omitempty" json:"cmd"`
	// ipv6匹配正则表达式
	IPv6Reg string   `yaml:",omitempty" json:"IPv6Reg"`
	Domains []string `yaml:",omitempty" json:"domains"`
}

// DDnsConfig 配置
type DDnsConfig struct {
	Name    string   `json:"name"`
	Delay   int64    `yaml:",omitempty" json:"delay"`
	Ipv4    *Ipv4    `yaml:",omitempty"  json:"ipv4"`
	Ipv6    *Ipv6    `yaml:",omitempty" json:"ipv6"`
	DNS     *DNS     `yaml:",omitempty" json:"dns"`
	TTL     string   `yaml:",omitempty" json:"ttl"`
	Webhook *Webhook `yaml:",omitempty" json:"webhook"`
}

func (conf *DDnsConfig) getIpv4AddrFromInterface() string {
	log := logger.Default()
	ipv4, _, err := util.GetNetInterface()
	if err != nil {
		log.Debugf("Failed to get IPv4 from network interface! error: %s\n", err.Error())
		return ""
	}

	for _, netInterface := range ipv4 {
		if netInterface.Name == conf.Ipv4.NetInterface && len(netInterface.Address) > 0 {
			return netInterface.Address[0]
		}
	}

	log.Debug("Failed to get IPv4 from network interface! Interface name: ", conf.Ipv4.NetInterface)
	return ""
}

func (conf *DDnsConfig) getIpv4AddrFromUrl() string {
	log := logger.Default()
	client := util.CreateNoProxyHTTPClient("tcp4")
	urls := strings.Split(conf.Ipv4.URL, ",")
	for _, url := range urls {
		url = strings.TrimSpace(url)
		resp, err := client.Get(url)
		if err != nil {
			log.Debugf("Error message: %s\n", err)
			continue
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Debugf("Failed to read IPv4 result! Interface:", url)
			continue
		}
		result := Ipv4Reg.FindString(string(body))
		if result == "" {
			log.Debugf("Failed to get IPv4 result! Interface: %s, return value: %s\n", url, result)
		}
		return result
	}
	return ""
}

func (conf *DDnsConfig) getAddrFromCmd(addrType string) string {
	log := logger.Default()
	var cmd string
	var comp *regexp.Regexp
	if addrType == "IPv4" {
		cmd = conf.Ipv4.Cmd
		comp = Ipv4Reg
	} else {
		cmd = conf.Ipv6.Cmd
		comp = Ipv6Reg
	}
	// cmd is empty
	if cmd == "" {
		return ""
	}
	// run cmd with proper shell
	var execCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		execCmd = exec.Command("powershell", "-Command", cmd)
	} else {
		// If Bash does not exist, use sh
		_, err := exec.LookPath("bash")
		if err != nil {
			execCmd = exec.Command("sh", "-c", cmd)
		} else {
			execCmd = exec.Command("bash", "-c", cmd)
		}
	}
	// run cmd
	out, err := execCmd.CombinedOutput()
	if err != nil {
		log.Debugf("Failed to get %s result! Failed to execute command: %s, error: %q, exit status code: %s\n", addrType, execCmd.String(), out, err)
		return ""
	}
	str := string(out)
	// get result
	result := comp.FindString(str)
	if result == "" {
		log.Debugf("Failed to get %s result! Command: %s, standard output: %q\n", addrType, execCmd.String(), str)
	}
	return result
}

// GetIpv4Addr 获得IPv4地址
func (conf *DDnsConfig) GetIpv4Addr() string {
	log := logger.Default()
	// 判断从哪里获取IP
	switch conf.Ipv4.GetType {
	case "netInterface":
		// 从网卡获取 IP
		return conf.getIpv4AddrFromInterface()
	case "url":
		// 从 URL 获取 IP
		return conf.getIpv4AddrFromUrl()
	case "cmd":
		// 从命令行获取 IP
		return conf.getAddrFromCmd("IPv4")
	default:
		log.Debugf("Unknown IPv4 way to get IP!")
		return "" // unknown type
	}
}

func (conf *DDnsConfig) getIpv6AddrFromInterface() string {
	log := logger.Default()
	_, ipv6, err := util.GetNetInterface()
	if err != nil {
		log.Debug("Failed to get IPv6 from network interface!")
		return ""
	}

	for _, netInterface := range ipv6 {
		if netInterface.Name == conf.Ipv6.NetInterface && len(netInterface.Address) > 0 {
			if conf.Ipv6.IPv6Reg != "" {
				// 匹配第几个IPv6
				if match, err := regexp.MatchString("@\\d", conf.Ipv6.IPv6Reg); err == nil && match {
					num, err := strconv.Atoi(conf.Ipv6.IPv6Reg[1:])
					if err == nil {
						if num > 0 {
							log.Debugf("IPv6 will use IPv6 address %d\n", num)
							if num <= len(netInterface.Address) {
								return netInterface.Address[num-1]
							}
							log.Debug("IPv6 address %d not found! First IPv6 address will be used\n", num)
							return netInterface.Address[0]
						}
						log.Debugf("IPv6 matching expression %s is incorrect! Minimum starts from 1\n", conf.Ipv6.IPv6Reg)
						return ""
					}
				}
				// 正则表达式匹配
				log.Infof("IPv6 will use the regular expression %s for matching\n", conf.Ipv6.IPv6Reg)
				for i := 0; i < len(netInterface.Address); i++ {
					matched, err := regexp.MatchString(conf.Ipv6.IPv6Reg, netInterface.Address[i])
					if matched && err == nil {
						log.Debugf("The match is successful! Matched to the address: ", netInterface.Address[i])
						return netInterface.Address[i]
					}
					log.Debugf("%d address %s does not match, will match next address\n", i+1, netInterface.Address[i])
				}
				log.Infof("Does not match any IPv6 address, the first address will be used")
			}
			return netInterface.Address[0]
		}
	}

	log.Infof("Failed to get IPv6 from network interface! Network interface name: ", conf.Ipv6.NetInterface)
	return ""
}

func (conf *DDnsConfig) getIpv6AddrFromUrl() string {
	log := logger.Default()
	client := util.CreateNoProxyHTTPClient("tcp6")
	urls := strings.Split(conf.Ipv6.URL, ",")
	for _, url := range urls {
		url = strings.TrimSpace(url)
		resp, err := client.Get(url)
		if err != nil {
			log.Infof("Error message: %s\n", err)
			continue
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Infof("Failed to read IPv6 result! Interface:", url)
			continue
		}
		result := Ipv6Reg.FindString(string(body))
		if result == "" {
			log.Infof("Failed to get IPv6 result! Interface: %s, return value: %s\n", url, result)
		}
		return result
	}
	return ""
}

// GetIpv6Addr 获得IPv6地址
func (conf *DDnsConfig) GetIpv6Addr() (result string) {
	log := logger.Default()
	// 判断从哪里获取IP
	switch conf.Ipv6.GetType {
	case "netInterface":
		// 从网卡获取 IP
		return conf.getIpv6AddrFromInterface()
	case "url":
		// 从 URL 获取 IP
		return conf.getIpv6AddrFromUrl()
	case "cmd":
		// 从命令行获取 IP
		return conf.getAddrFromCmd("IPv6")
	default:
		log.Infof("Unknown way to get IP for IPv6!")
		return "" // unknown type
	}
}

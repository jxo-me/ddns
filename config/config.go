package config

import (
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"sync"
)

const ConfigFilePathENV = "DDNS_CONFIG_FILE_PATH"

var cache = &cacheType{}

// ConfigCache ConfigCache
type cacheType struct {
	ConfigSingle *Config
	Err          error
	Lock         sync.Mutex
}

type Config struct {
	DnsConf []DnsConfig
	Webhook
	// 禁止公网访问
	NotAllowWanAccess bool
}

// CompatibleConfig 兼容v5.0.0之前的配置文件
func (conf *Config) CompatibleConfig() {
	if len(conf.DnsConf) > 0 {
		return
	}

	configFilePath := GetConfigFilePath()
	_, err := os.Stat(configFilePath)
	if err != nil {
		return
	}
	byt, err := os.ReadFile(configFilePath)
	if err != nil {
		return
	}

	dnsConf := &DnsConfig{}
	err = yaml.Unmarshal(byt, dnsConf)
	if err != nil {
		return
	}
	if len(dnsConf.DNS.Name) > 0 {
		cache.Lock.Lock()
		defer cache.Lock.Unlock()
		conf.DnsConf = append(conf.DnsConf, *dnsConf)
		cache.ConfigSingle = conf
	}
}

// SaveConfig 保存配置
func (conf *Config) SaveConfig() (err error) {
	cache.Lock.Lock()
	defer cache.Lock.Unlock()

	byt, err := yaml.Marshal(conf)
	if err != nil {
		log.Println(err)
		return err
	}

	configFilePath := GetConfigFilePath()
	err = os.WriteFile(configFilePath, byt, 0600)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf("配置文件已保存在: %s\n", configFilePath)

	// 清空配置缓存
	cache.ConfigSingle = nil

	return
}

// GetConfigCached 获得缓存的配置
func GetConfigCached() (conf Config, err error) {
	cache.Lock.Lock()
	defer cache.Lock.Unlock()

	if cache.ConfigSingle != nil {
		return *cache.ConfigSingle, cache.Err
	}

	// init config
	cache.ConfigSingle = &Config{}

	configFilePath := GetConfigFilePath()
	_, err = os.Stat(configFilePath)
	if err != nil {
		cache.Err = err
		return *cache.ConfigSingle, err
	}

	byt, err := os.ReadFile(configFilePath)
	if err != nil {
		log.Println(configFilePath + " 读取失败")
		cache.Err = err
		return *cache.ConfigSingle, err
	}

	err = yaml.Unmarshal(byt, cache.ConfigSingle)
	if err != nil {
		log.Println("反序列化配置文件失败", err)
		cache.Err = err
		return *cache.ConfigSingle, err
	}

	// remove err
	cache.Err = nil
	return *cache.ConfigSingle, err
}

// GetConfigFilePath 获得配置文件路径
func GetConfigFilePath() string {
	configFilePath := os.Getenv(ConfigFilePathENV)
	if configFilePath != "" {
		return configFilePath
	}
	return GetConfigFilePathDefault()
}

// GetConfigFilePathDefault 获得默认的配置文件路径
func GetConfigFilePathDefault() string {
	dir, err := os.UserHomeDir()
	if err != nil {
		// log.Println("Getting Home directory failed: ", err)
		return "../.ddns_go_config.yaml"
	}
	return dir + string(os.PathSeparator) + ".ddns_go_config.yaml"
}

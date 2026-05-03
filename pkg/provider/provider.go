package provider

import (
	"fmt"
	"gd-agent/pkg/common"
	"gd-agent/pkg/llms"
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type ProviderInfo struct {
	BaseUrl      string   `yaml:"base_url"`
	ApiKey       string   `yaml:"api_key"`
	ModelList    []string `yaml:"model_list"`
	DefaultModel string   `yaml:"default_model"`
}

// 服务提供商接口类型
type Provider interface {
	GetResponse(state *llms.State) (<-chan rune, error)
}

var providerInfoList = make(map[string]ProviderInfo)
var ProviderList map[string]Provider
var AvailableProviderNames []string

type ProviderFactory func(ProviderInfo) Provider

var providerFactories = make(map[string]ProviderFactory)

func init() {
	data, readFileErr := os.ReadFile("config/providers.yaml")
	if readFileErr != nil {
		panic(fmt.Sprintf("read provider config error[%d]: %v", common.ERROR_Init, readFileErr))
	}

	if yamlUnmarshalErr := yaml.Unmarshal(data, &providerInfoList); yamlUnmarshalErr != nil {
		panic(fmt.Sprintf(
			"parse provider config error[%d]: %v", common.ERROR_Init, yamlUnmarshalErr,
		))
	}
	// find api key in .env file
	loadDotEnvErr := godotenv.Load()
	if loadDotEnvErr != nil {
		panic(fmt.Errorf("load .env file to read api key error[%d]: %v", common.ERROR_Init, loadDotEnvErr))
	}

	for providerName, providerInfo := range providerInfoList {
		AvailableProviderNames = append(AvailableProviderNames, providerName)
		providerInfo.ApiKey = os.Getenv(providerInfo.ApiKey)
		providerInfoList[providerName] = providerInfo
	}
}

func Register(name string, factory ProviderFactory) {
	providerFactories[name] = factory
}

func InitProviderList() {
	ProviderList = make(map[string]Provider)
	for name, info := range providerInfoList {
		if factory, ok := providerFactories[name]; ok {
			ProviderList[name] = factory(info)
		}
	}
}

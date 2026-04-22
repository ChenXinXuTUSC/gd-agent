package llms

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"gd-agent/pkg/common"
	llm_types "gd-agent/pkg/llms/types"

	p_deepseek "gd-agent/pkg/llms/deepseek"

	"github.com/joho/godotenv"
)

func init() {
	var providerInfoList map[string]llm_types.ProviderInfo
	data, readFileErr := os.ReadFile("config/providers.yaml")
	if readFileErr != nil {
		panic(fmt.Sprintf("read provider config error[%d]: %v", common.ERROR_Init, readFileErr))
	}
	providerInfoList = make(map[string]llm_types.ProviderInfo)
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

	// fmt.Printf("%+v\n", providerInfoList)

	ProviderList = make(map[string]Provider)
	ProviderList["DeepSeek"] = p_deepseek.ProviderDeepSeek{Info: providerInfoList["DeepSeek"]}
}

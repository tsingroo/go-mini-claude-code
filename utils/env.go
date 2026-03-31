package utils

import (
	"errors"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
)

type ModelInfo struct {
	BaseUrl   string
	ApiKey    string
	ModelName string
}

func GetModelInfo() (*ModelInfo, error) {
	BASE_URL, hasBaseUrl := os.LookupEnv("ANTHROPIC_BASEURL")
	API_KEY, hasKey := os.LookupEnv("ANTHROPIC_KEY")
	MODEL_NAME, hasModelName := os.LookupEnv("ANTHROPIC_MODEL")

	if !hasBaseUrl || !hasKey {
		return nil, errors.New("请使用 export ANTHROPIC_BASEURL=\"xxxx\" 或者export ANTHROPIC_KEY=\"yyy\" 设置环境变量")
	}
	if !hasModelName {
		MODEL_NAME = string(anthropic.ModelClaudeOpus4_6)
	}

	return &ModelInfo{
		BaseUrl:   BASE_URL,
		ApiKey:    API_KEY,
		ModelName: MODEL_NAME,
	}, nil
}

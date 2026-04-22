package deepseek

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	. "gd-agent/pkg/common"
	llm_types "gd-agent/pkg/llms/types"

	"io"
	"net/http"
	"time"
)

type ProviderDeepSeek struct {
	Info llm_types.ProviderInfo
}

// DeepSeek 相关请求与响应格式定义
type ChatReq struct {
	Model    string              `json:"model"`
	Messages []llm_types.Message `json:"messages"`
	Stream   bool                `json:"stream"`
}

// 非流式响应
type ChatResp struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
	SystemFingerprint string   `json:"system_fingerprint"`
}
type Choice struct {
	Index        int               `json:"index"`
	Message      llm_types.Message `json:"message"`
	Logprobs     interface{}       `json:"logprobs"`
	FinishReason string            `json:"finish_reason"`
}
type Usage struct {
	PromptTokens          int                 `json:"prompt_tokens"`
	CompletionTokens      int                 `json:"completion_tokens"`
	TotalTokens           int                 `json:"total_tokens"`
	PromptTokensDetails   PromptTokensDetails `json:"prompt_tokens_details"`
	PromptCacheHitTokens  int                 `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens int                 `json:"prompt_cache_miss_tokens"`
}
type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

// 流式响应输出
type StreamChunk struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []StreamChoice `json:"choices"`
}

type StreamChoice struct {
	Index        int     `json:"index"`
	Delta        Delta   `json:"delta"`
	FinishReason *string `json:"finish_reason"` // 用指针，因为大多数帧是 null
}

type Delta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content"`
}

func (p ProviderDeepSeek) GetResponse(state *llm_types.State) (<-chan rune, error) {
	resp, err := p.doHttpRequest(state.Messages, state.Stream)
	if err != nil {
		return nil, WrapErr(err)
	}

	var runeCh <-chan rune = nil
	var respErr error
	if state.Stream {
		runeCh, respErr = p.parseStreamResp(resp)
	} else {
		runeCh, respErr = p.parseNoneStreamResp(resp)
	}

	return runeCh, respErr
}

func (p ProviderDeepSeek) doHttpRequest(reqMsgs []llm_types.Message, streamMode bool) (*http.Response, error) {
	reqBody, err := json.Marshal(ChatReq{
		Model:    p.Info.DefaultModel,
		Messages: reqMsgs,
		Stream:   streamMode,
	})
	if err != nil {
		return nil, WrapErr(err)
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second, // 仅控制 TCP 连接超时
		}).DialContext,
		ResponseHeaderTimeout: 15 * time.Second, // 仅控制等待响应头超时
	}

	req, err := http.NewRequest("POST", p.Info.BaseUrl+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, WrapErr(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.Info.ApiKey)

	client := &http.Client{Transport: transport} // 不设 Timeout，Body 读取不受限
	resp, err := client.Do(req)
	if err != nil {
		return nil, WrapErr(err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, WrapErr(fmt.Errorf("http status %d", resp.StatusCode))
	}

	return resp, nil
}

func (p ProviderDeepSeek) parseNoneStreamResp(resp *http.Response) (<-chan rune, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, WrapErr(fmt.Errorf("read http response error: %w", err))
	}

	var result = ChatResp{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, WrapErr(fmt.Errorf("parse resp failed: %w, body(%d): %s", err, len(body), string(body)))
	}
	if len(result.Choices) == 0 {
		return nil, WrapErr(fmt.Errorf("empty choices"))
	}

	content := result.Choices[0].Message.Content
	ch := make(chan rune, len([]rune(content)))
	for _, r := range content {
		ch <- r
	}
	close(ch)

	return ch, nil
}

func (p ProviderDeepSeek) parseStreamResp(resp *http.Response) (<-chan rune, error) {
	ch := make(chan rune, 512)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

		for scanner.Scan() {
			// if err := scanner.Err(); err != nil {
			// 	fmt.Println("scanner error:", err)
			// }

			line := scanner.Text()

			if line == "" {
				continue
			}
			if line == "data: [DONE]" {
				break
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			jsonStr := strings.TrimPrefix(line, "data: ")

			var chunk = StreamChunk{}
			if err := json.Unmarshal([]byte(jsonStr), &chunk); err != nil {
				fmt.Printf("%v", WrapErr(err))
			}
			if len(chunk.Choices) == 0 {
				continue
			}

			for _, r := range chunk.Choices[0].Delta.Content {
				ch <- r
			}
		}
	}()

	return ch, nil
}

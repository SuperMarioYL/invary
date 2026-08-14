// Package model holds the OpenAI-compatible provider registry and (in m2) the
// hand-rolled HTTP client that drives all four CN providers with a single
// shared round-tripper.
package model

// Provider is one OpenAI-compatible CN model endpoint. All four v0.1 providers
// expose chat-completions-with-tools at an OpenAI-compatible base URL, so the
// client swaps only baseURL + apiKey + modelID per provider — no per-provider
// SDK, no version drift.
type Provider struct {
	Name     string // canonical id: deepseek | qwen | kimi | glm
	Label    string // human-readable label
	BaseURL  string // OpenAI-compatible base URL (no trailing slash)
	ModelID  string // default chat model id for the differential run
	APIKeyEnv string // environment variable holding the provider api key
}

// Providers is the built-in registry of the four v0.1 CN providers, keyed by
// canonical id. These are static configuration values only — no network call
// happens until the m2 client is implemented.
var Providers = map[string]Provider{
	"deepseek": {
		Name:      "deepseek",
		Label:     "DeepSeek",
		BaseURL:   "https://api.deepseek.com/v1",
		ModelID:   "deepseek-chat",
		APIKeyEnv: "DEEPSEEK_API_KEY",
	},
	"qwen": {
		Name:      "qwen",
		Label:     "Qwen",
		BaseURL:   "https://dashscope.aliyuncs.com/compatible-mode/v1",
		ModelID:   "qwen-plus",
		APIKeyEnv: "QWEN_API_KEY",
	},
	"kimi": {
		Name:      "kimi",
		Label:     "Kimi",
		BaseURL:   "https://api.moonshot.cn/v1",
		ModelID:   "moonshot-v1-8k",
		APIKeyEnv: "KIMI_API_KEY",
	},
	"glm": {
		Name:      "glm",
		Label:     "GLM",
		BaseURL:   "https://open.bigmodel.cn/api/paas/v4",
		ModelID:   "glm-4",
		APIKeyEnv: "GLM_API_KEY",
	},
}

// Order is the canonical evaluation order for the differential run.
var Order = []string{"deepseek", "qwen", "kimi", "glm"}

// ByName looks up a provider by canonical id.
func ByName(name string) (Provider, bool) {
	p, ok := Providers[name]
	return p, ok
}

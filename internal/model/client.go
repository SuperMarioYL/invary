package model

import "fmt"

// Client is the hand-rolled OpenAI-compatible HTTP client. In m2 it sends a
// chat-completions-with-tools request to the provider's base URL using only
// net/http (no per-provider SDK), then returns the raw tool-call trace for the
// invariant evaluator to reason about. m1 ships this as a stub so the package
// shape is in place; the live implementation lands in milestone m2.
type Client struct {
	Provider Provider
	APIKey  string
}

// Call sends the prompt + tool schema to the provider and returns the raw
// chat-completions response bytes (the captured trace).
func (c *Client) Call(prompt string, schema []byte) ([]byte, error) {
	return nil, fmt.Errorf("model client not implemented until milestone m2")
}

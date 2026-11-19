package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func serveWithAlias(t *testing.T, requested string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/x", ResponseModelAlias(), func(c *gin.Context) {
		if requested != "" {
			c.Set("original_model", requested) // what Distribute records
		}
		c.Next()
	}, handler)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("{}")))
	return w
}

// Mirrors service.IOCopyBytesGracefully: Content-Length from the upstream body.
func writeUpstreamJSON(c *gin.Context, body string) {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	c.Writer.WriteHeader(http.StatusOK)
	_, _ = c.Writer.Write([]byte(body))
}

func TestResponseModelAliasNonStream(t *testing.T) {
	cases := map[string]struct{ body, path string }{
		"chat":       {`{"id":"c1","object":"chat.completion","model":"deepseek-v4.1-flash","choices":[{"message":{"role":"assistant","content":"\"model\":\"x\""}}],"usage":{"total_tokens":3}}`, "model"},
		"embeddings": {`{"object":"list","data":[{"embedding":[0.1]}],"model":"bge-m3-upstream","usage":{"prompt_tokens":1}}`, "model"},
		"responses":  {`{"id":"resp_1","object":"response","model":"deepseek-v4.1-flash","output":[]}`, "model"},
		"claude":     {`{"id":"msg_1","type":"message","model":"claude-upstream","content":[]}`, "model"},
		"gemini":     {`{"candidates":[],"modelVersion":"gemini-upstream-001"}`, "modelVersion"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			w := serveWithAlias(t, "kimi-k3", func(c *gin.Context) { writeUpstreamJSON(c, tc.body) })
			body := w.Body.Bytes()
			if got := gjson.GetBytes(body, tc.path).Str; got != "kimi-k3" {
				t.Fatalf("%s = %q, want kimi-k3; body %s", tc.path, got, body)
			}
			if cl := w.Header().Get("Content-Length"); cl != strconv.Itoa(len(body)) {
				t.Fatalf("Content-Length %s, body length %d", cl, len(body))
			}
			if name == "chat" {
				if got := gjson.GetBytes(body, "choices.0.message.content").Str; got != `"model":"x"` {
					t.Fatalf("message content changed: %q", got)
				}
				if gjson.GetBytes(body, "usage.total_tokens").Int() != 3 {
					t.Fatalf("usage changed: %s", body)
				}
			}
		})
	}
}

func TestResponseModelAliasStreamOpenAIChunks(t *testing.T) {
	w := serveWithAlias(t, "kimi-k3", func(c *gin.Context) {
		helper.SetEventStreamHeaders(c)
		for i := 0; i < 3; i++ {
			_ = helper.StringData(c, `{"id":"c1","object":"chat.completion.chunk","model":"deepseek-v4.1-flash","choices":[{"delta":{"content":"hi"}}]}`)
		}
		_ = helper.StringData(c, `{"id":"c1","object":"chat.completion.chunk","model":"deepseek-v4.1-flash","choices":[],"usage":{"total_tokens":9}}`)
		helper.Done(c)
	})
	events := 0
	for _, line := range strings.Split(w.Body.String(), "\n") {
		payload, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}
		if payload == "[DONE]" {
			continue
		}
		events++
		if got := gjson.Get(payload, "model").Str; got != "kimi-k3" {
			t.Fatalf("chunk model %q: %s", got, payload)
		}
	}
	if events != 4 || !strings.Contains(w.Body.String(), "data: [DONE]") {
		t.Fatalf("unexpected stream: %q", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "deepseek") {
		t.Fatalf("upstream name leaked: %s", w.Body.String())
	}
}

func TestResponseModelAliasStreamClaudeResponsesGemini(t *testing.T) {
	// Raw passthrough writes, split mid-line and with CRLF line ends.
	stream := "event: message_start\r\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"m\",\"model\":\"claude-upstream\",\"usage\":{\"input_tokens\":5}}}\r\n\r\n" +
		"event: response.created\ndata: {\"type\":\"response.created\",\"response\":{\"id\":\"r\",\"model\":\"deepseek-v4.1-flash\"}}\n\n" +
		"data: {\"candidates\":[],\"modelVersion\":\"gemini-upstream\"}\n\n" +
		"data:{\"model\":\"no-space-upstream\"}\n\n"
	w := serveWithAlias(t, "kimi-k3", func(c *gin.Context) {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		for i := 0; i < len(stream); i += 7 {
			end := min(i+7, len(stream))
			_, _ = c.Writer.Write([]byte(stream[i:end]))
		}
		c.Writer.Flush()
	})
	out := w.Body.String()
	for _, leaked := range []string{"claude-upstream", "deepseek", "gemini-upstream", "no-space-upstream"} {
		if strings.Contains(out, leaked) {
			t.Fatalf("%s leaked: %q", leaked, out)
		}
	}
	if strings.Count(out, "kimi-k3") != 4 || !strings.Contains(out, "event: message_start\r\ndata: {") || !strings.Contains(out, "data:{\"model\":\"kimi-k3\"}") {
		t.Fatalf("unexpected rewritten stream: %q", out)
	}
	if !strings.Contains(out, `"usage":{"input_tokens":5}`) {
		t.Fatalf("usage changed: %q", out)
	}
}

func TestResponseModelAliasPassthrough(t *testing.T) {
	body := `{"model":"deepseek-v4.1-flash"}`
	// No requested model known (e.g. rejected before channel selection).
	if w := serveWithAlias(t, "", func(c *gin.Context) { writeUpstreamJSON(c, body) }); w.Body.String() != body {
		t.Fatalf("changed without requested model: %s", w.Body.String())
	}
	// Binary bodies are untouched.
	audio := []byte("ID3\x00\"model\":\"x\"")
	w := serveWithAlias(t, "tts-1", func(c *gin.Context) { c.Data(http.StatusOK, "audio/mpeg", audio) })
	if !bytes.Equal(w.Body.Bytes(), audio) {
		t.Fatalf("audio changed")
	}
	// Toggle off.
	ResponseModelAliasEnabled = false
	t.Cleanup(func() { ResponseModelAliasEnabled = true })
	if w := serveWithAlias(t, "kimi-k3", func(c *gin.Context) { writeUpstreamJSON(c, body) }); w.Body.String() != body {
		t.Fatalf("changed while disabled: %s", w.Body.String())
	}
}

func TestResponseModelAliasErrorAfterBufferedBody(t *testing.T) {
	// Relay code checks Written() before emitting an error body; a buffered
	// response must count as written so no second body is appended.
	w := serveWithAlias(t, "kimi-k3", func(c *gin.Context) {
		writeUpstreamJSON(c, `{"model":"up","choices":[]}`)
		if !c.Writer.Written() {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "late"})
		}
	})
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), "late") || gjson.Get(w.Body.String(), "model").Str != "kimi-k3" {
		t.Fatalf("unexpected response %d %s", w.Code, w.Body.String())
	}
}

func TestResponseModelAliasUpstreamQuotaExtension(t *testing.T) {
	body := `{"model":"deepseek-v4.1-flash","choices":[],"usage":{"total_tokens":8,"quota":{"model":"deepseek-v4.1-flash","percent":0,"shared_models":["deepseek-v4-pro"]}}}`
	w := serveWithAlias(t, "kimi-k3", func(c *gin.Context) { writeUpstreamJSON(c, body) })
	out := w.Body.String()
	if strings.Contains(out, "deepseek") {
		t.Fatalf("upstream name leaked: %s", out)
	}
	if gjson.Get(out, "usage.quota.model").Str != "kimi-k3" || gjson.Get(out, "usage.total_tokens").Int() != 8 {
		t.Fatalf("unexpected usage: %s", out)
	}
	empty := `{"model":"up","usage":{"quota":{"model":"up","shared_models":[]}}}`
	w = serveWithAlias(t, "kimi-k3", func(c *gin.Context) { writeUpstreamJSON(c, empty) })
	if !gjson.Get(w.Body.String(), "usage.quota.shared_models").IsArray() {
		t.Fatalf("empty shared_models should stay: %s", w.Body.String())
	}
}

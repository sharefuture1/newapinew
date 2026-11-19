// Copyright (C) 2023-2026 QuantumNous. Modified by WendingAI (2026-11):
// relay responses report the model name the client asked for.
package middleware

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ResponseModelAliasEnabled controls whether relay responses are rewritten so
// that their model fields carry the model name from the client's request
// instead of the name declared by the upstream. Defaults to on; set
// RESPONSE_MODEL_ALIAS_ENABLED=false to pass upstream names through.
var ResponseModelAliasEnabled = common.GetEnvOrDefaultBool("RESPONSE_MODEL_ALIAS_ENABLED", true)

// responseModelAliasPaths are the JSON locations that declare the serving model
// in OpenAI (chat, completions, embeddings, images, responses), Claude and
// Gemini payloads, both for whole bodies and for individual stream events.
var responseModelAliasPaths = []string{"model", "message.model", "response.model", "modelVersion"}

// ResponseModelAlias wraps the response writer for relay routes. It only
// changes what the client receives: billing, logs and diagnostics read the
// upstream body before it is written and are unaffected. Install it before
// Distribute so the requested model is known by the time the body is written.
func ResponseModelAlias() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ResponseModelAliasEnabled {
			c.Next()
			return
		}
		original := c.Writer
		w := &modelAliasWriter{ResponseWriter: original, c: c}
		c.Writer = w
		defer func() {
			w.finish()
			if c.Writer == w {
				c.Writer = original
			}
		}()
		c.Next()
	}
}

const (
	aliasModeUndecided = iota
	aliasModePassthrough
	aliasModeJSON  // buffered whole body, rewritten when the handler returns
	aliasModeLines // SSE / NDJSON, rewritten line by line
)

type modelAliasWriter struct {
	gin.ResponseWriter
	c       *gin.Context
	mode    int
	sse     bool
	target  string
	buf     bytes.Buffer // JSON body, or the incomplete trailing line
	written bool         // something was accepted from the handler
}

func (w *modelAliasWriter) decide() {
	if w.mode != aliasModeUndecided {
		return
	}
	w.mode = aliasModePassthrough
	w.target = w.c.GetString(string(constant.ContextKeyOriginalModel))
	if w.target == "" {
		return
	}
	header := w.ResponseWriter.Header()
	if enc := header.Get("Content-Encoding"); enc != "" && !strings.EqualFold(enc, "identity") {
		return
	}
	contentType := strings.ToLower(header.Get("Content-Type"))
	switch {
	case strings.Contains(contentType, "text/event-stream"):
		w.mode, w.sse = aliasModeLines, true
	case strings.Contains(contentType, "ndjson"):
		w.mode = aliasModeLines
	case strings.Contains(contentType, "json"):
		w.mode = aliasModeJSON
	}
}

func (w *modelAliasWriter) Write(p []byte) (int, error) {
	w.decide()
	switch w.mode {
	case aliasModeJSON:
		w.written = true
		return w.buf.Write(p)
	case aliasModeLines:
		w.written = true
		w.buf.Write(p)
		if err := w.emitCompleteLines(); err != nil {
			return 0, err
		}
		return len(p), nil
	default:
		return w.ResponseWriter.Write(p)
	}
}

func (w *modelAliasWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *modelAliasWriter) Written() bool {
	return w.written || w.ResponseWriter.Written()
}

func (w *modelAliasWriter) Size() int {
	if w.mode == aliasModeJSON && w.buf.Len() > 0 && !w.ResponseWriter.Written() {
		return w.buf.Len()
	}
	return w.ResponseWriter.Size()
}

// WriteHeaderNow is deferred for buffered JSON bodies: their Content-Length is
// only known once the rewritten body is ready.
func (w *modelAliasWriter) WriteHeaderNow() {
	w.decide()
	if w.mode == aliasModeJSON {
		return
	}
	w.ResponseWriter.WriteHeaderNow()
}

func (w *modelAliasWriter) Flush() {
	switch w.mode {
	case aliasModeJSON:
		return
	case aliasModeLines:
		_ = w.emitCompleteLines()
		if w.buf.Len() > 0 {
			// A writer flushed mid-line; do not hold the bytes back.
			_, _ = w.ResponseWriter.Write(w.buf.Bytes())
			w.buf.Reset()
		}
	}
	w.ResponseWriter.Flush()
}

func (w *modelAliasWriter) emitCompleteLines() error {
	data := w.buf.Bytes()
	last := bytes.LastIndexByte(data, '\n')
	if last < 0 {
		return nil
	}
	var out bytes.Buffer
	out.Grow(last + 1)
	rest := data[:last+1]
	for len(rest) > 0 {
		i := bytes.IndexByte(rest, '\n')
		out.Write(w.rewriteLine(rest[:i]))
		out.WriteByte('\n')
		rest = rest[i+1:]
	}
	remaining := append([]byte(nil), data[last+1:]...)
	w.buf.Reset()
	w.buf.Write(remaining)
	_, err := w.ResponseWriter.Write(out.Bytes())
	return err
}

func (w *modelAliasWriter) rewriteLine(line []byte) []byte {
	body := bytes.TrimSuffix(line, []byte("\r"))
	suffix := line[len(body):]
	if w.sse {
		if !bytes.HasPrefix(body, []byte("data:")) {
			return line
		}
		payload := body[len("data:"):]
		lead := len(payload) - len(bytes.TrimLeft(payload, " "))
		rewritten := rewriteResponseModel(payload[lead:], w.target)
		if rewritten == nil {
			return line
		}
		out := make([]byte, 0, len(line)+len(w.target))
		out = append(out, body[:len("data:")+lead]...)
		out = append(out, rewritten...)
		return append(out, suffix...)
	}
	rewritten := rewriteResponseModel(body, w.target)
	if rewritten == nil {
		return line
	}
	return append(rewritten, suffix...)
}

func (w *modelAliasWriter) finish() {
	switch w.mode {
	case aliasModeJSON:
		if w.buf.Len() == 0 {
			return
		}
		body := w.buf.Bytes()
		if rewritten := rewriteResponseModel(body, w.target); rewritten != nil {
			body = rewritten
		}
		w.buf.Reset()
		if !w.ResponseWriter.Written() {
			w.ResponseWriter.Header().Set("Content-Length", strconv.Itoa(len(body)))
		}
		_, _ = w.ResponseWriter.Write(body)
	case aliasModeLines:
		if w.buf.Len() > 0 {
			line := w.rewriteLine(w.buf.Bytes())
			w.buf.Reset()
			_, _ = w.ResponseWriter.Write(line)
		}
	}
}

// rewriteResponseModel returns body with every declared model replaced by
// target, or nil when nothing needs to change (including non-JSON payloads
// such as "[DONE]").
func rewriteResponseModel(body []byte, target string) []byte {
	if len(body) == 0 || target == "" || !bytes.Contains(body, []byte(`"model`)) {
		return nil
	}
	if !gjson.ValidBytes(body) {
		return nil
	}
	out := body
	changed := false
	for _, path := range responseModelAliasPaths {
		value := gjson.GetBytes(out, path)
		if value.Type != gjson.String || value.Str == "" || value.Str == target {
			continue
		}
		next, err := sjson.SetBytes(out, path, target)
		if err != nil {
			continue
		}
		out, changed = next, true
	}
	if !changed {
		return nil
	}
	return out
}

// Unwrap lets http.ResponseController reach the underlying writer (stream
// write deadlines).
func (w *modelAliasWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

var _ http.Flusher = (*modelAliasWriter)(nil)

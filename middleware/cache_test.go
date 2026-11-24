package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCacheHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(Cache())
	router.GET("/*path", func(c *gin.Context) { c.Status(http.StatusOK) })

	cases := []struct {
		path string
		want string
	}{
		{"/", "no-cache"},
		{"/static/js/index.abc123.js", "public, max-age=31536000, immutable"},
		{"/static/css/index.abc123.css", "public, max-age=31536000, immutable"},
		{"/logo.png", "max-age=604800"},
		{"/favicon.ico", "max-age=604800"},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		router.ServeHTTP(rec, req)
		assert.Equal(t, tc.want, rec.Header().Get("Cache-Control"), tc.path)
		assert.NotEmpty(t, rec.Header().Get("Cache-Version"), tc.path)
	}
}

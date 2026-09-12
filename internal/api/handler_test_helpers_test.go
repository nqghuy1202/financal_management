package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// newTestHandler builds a Handler wired to a sqlmock-backed *sql.DB, so every
// repo it owns runs against the same mock expectations.
func newTestHandler(t *testing.T) (*Handler, sqlmock.Sqlmock, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	h := NewHandler(db, []byte("test-secret"))
	return h, mock, func() { db.Close() }
}

// authedRequest builds an httptest recorder + gin context for method/path
// with userID already set in the context (as AuthMiddleware would have set
// it), optionally with a JSON body and URL params.
func authedRequest(method, path, body, userID string, params gin.Params) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	c.Request = req
	c.Params = params
	c.Set(ctxUserID, userID)
	return w, c
}

// plainRequest is like authedRequest but does not set ctxUserID — used for
// exercising AuthMiddleware/full-handler flows directly.
func plainRequest(method, path, body string, headers map[string]string) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	return w, c
}

// ginParams builds a single-entry gin.Params, e.g. for handlers reading
// c.Param("id").
func ginParams(key, value string) gin.Params {
	return gin.Params{{Key: key, Value: value}}
}

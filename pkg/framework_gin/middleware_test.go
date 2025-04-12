package framework_gin

import (
    "net/http"
    "net/http/httptest"
    "os"
    "testing"

    "github.com/abstratium-informatique-sarl/stratis/pkg/jwt"
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {

    code := m.Run() // Run all tests in the package

    // Teardown code (e.g., stop the server, clean up the database)

    os.Exit(code)
}

func TestSecurityMiddleware_wrongRoles(t *testing.T) {
    assert := assert.New(t)

    sut := SecurityMiddleware([]string{"role1"})

    // https://stackoverflow.com/questions/41742988/make-mock-gin-context
    gin.SetMode(gin.TestMode)
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Set("user", &jwt.User{
        Username: "john.smith", 
        UserId: "1", 
        Expires: 0, 
        Roles: []string{"wrongOne"}, 
        UserContext: map[string]string{},
    })

    // when
    sut(c)

    // then
    assert.Equal(http.StatusForbidden, w.Code)
}

func TestSecurityMiddleware_anonymous(t *testing.T) {
    assert := assert.New(t)

    sut := SecurityMiddleware([]string{"role1"})
    gin.SetMode(gin.TestMode)
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Set("user", &jwt.User{
        Username: jwt.ANONYMOUS, 
        UserId: "0", 
        Expires: 0, 
        Roles: []string{}, 
        UserContext: map[string]string{},
    })

    // when
    sut(c)

    // then
    assert.Equal(http.StatusUnauthorized, w.Code)
}

func TestSecurityMiddleware_rightRoles(t *testing.T) {
    assert := assert.New(t)

    sut := SecurityMiddleware([]string{"role1"})
    gin.SetMode(gin.TestMode)
    w := httptest.NewRecorder()
    c, _ := gin.CreateTestContext(w)
    c.Set("user", &jwt.User{
        Username: "john.smith", 
        UserId: "0", 
        Expires: 0, 
        Roles: []string{"role1", "role2"}, 
        UserContext: map[string]string{},
    })

    // when
    sut(c)

    // then
    assert.Equal(http.StatusOK, w.Code)
}

package fwctx

import (
	"time"

	"github.com/abstratium-informatique-sarl/stratis/pkg/jwt"
	"github.com/abstratium-informatique-sarl/stratis/pkg/logging"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

var user jwt.User = jwt.User{
	Username: "some.one@dot.com", 
	UserId: "c606cb7f-9ac5-4f10-8403-db2e83b1ae0f", 
	Expires: float64(time.Now().UnixMilli()+(time.Duration.Milliseconds(60_000))), 
	Roles: []string{"testrole"}, 
	UserContext: map[string]string{},
}

type testCtx struct {
	db *gorm.DB
	isRollbackOnly bool
	isTransactional bool
}

func (c *testCtx) SetDb(db *gorm.DB, isTransactional bool) {
	c.db = db
	c.isTransactional = isTransactional
}

func (c *testCtx) GetDb() *gorm.DB {
	if c.db == nil {
		panic("call SetDb for this test")
	}
	return c.db
}

// returns true, if #SetRollbackOnly() has been called
func (c *testCtx) IsRollbackOnly() bool {
	return c.isRollbackOnly
}

// sets up the transaction to be rolled back
func (c *testCtx) SetRollbackOnly() {
	if c.db == nil {
		// ignore
	} else {
		if !c.isTransactional {
			panic("unable to rollback non-transactional connection")
		} else {
			c.isRollbackOnly = true
		}
	}
}

func (c *testCtx) GetGinCtx() *gin.Context {
	return nil
}

func (c *testCtx) QueryParamAsBooleanWithDefault(name string, defaultValue bool) bool {
	return false
}

func (c *testCtx) QueryParamAsString(name string) string {
	return ""
}

func (c *testCtx) QueryParamAsInt(name string) (int, error) {
	return 0, nil
}

func (c *testCtx) UnmarshalRequestBody(a any) error {
	panic("not supported yet")
}

func (c *testCtx) GetUser() (*jwt.User, error) {
	return &user, nil
}

func (c *testCtx) UserHasARole(rolesAllowed []string) (bool, error) {
	user, err := c.GetUser()
	if err != nil {
		panic("how did that happen?")
	}
	return UserHasARole(rolesAllowed, user), nil
}

func (c *testCtx) Debug(format string, a ...any) {
	packageName, _/*funcName*/ := getCallerInfo(0)
	l := logging.GetLog(packageName).With().Logger()
	l.Debug().Msgf(format, a...)
}

func (c *testCtx) Info(format string, a ...any) {
	packageName, _/*funcName*/ := getCallerInfo(0)
	l := logging.GetLog(packageName).With().Logger()
	l.Debug().Msgf(format, a...)
}

func (c *testCtx) Warn(format string, a ...any) {
	packageName, _/*funcName*/ := getCallerInfo(0)
	l := logging.GetLog(packageName).With().Logger()
	l.Debug().Msgf(format, a...)
}

func (c *testCtx) Error(format string, a ...any) {
	packageName, _/*funcName*/ := getCallerInfo(0)
	l := logging.GetLog(packageName).With().Logger()
	l.Debug().Msgf(format, a...)
}

func (c *testCtx) HandleError(err error, msg string, log zerolog.Logger) {
	c.SetRollbackOnly()
	log.Warn().Msgf("%s: %+v", msg, err)
}

func (c *testCtx) StartSpan(name string, isRemote bool) trace.Span {
	return nil
}

func BuildTypedCtxForTests(db *gorm.DB, isTransactional bool) ICtx {
	var ctx = &testCtx{db, false, isTransactional}
	return ICtx(ctx)
}

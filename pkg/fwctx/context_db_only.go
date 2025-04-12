package fwctx

// used for when stuff is done from a ticker, i.e. no user interaction

import (
	"context"

	"github.com/abstratium-informatique-sarl/stratis/pkg/jwt"
	"github.com/abstratium-informatique-sarl/stratis/pkg/logging"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type ctxWithOnlyDb struct {
	id              int
	db              *gorm.DB
	rollbackOnly    bool
	isTransactional bool
	username        string
	userId          string
	roles           []string
}

// designed for using for background tasks with a service user and in combination with database.WithTx() or database.NoTx()
func BuildTypedCtxNoDbNoGin(username string, userId string, roles []string) ICtx {
	id := highestId
	highestId++

	var context = &ctxWithOnlyDb{id, nil, false, true, username, userId, roles}
	var ictx = context
	return ICtx(ictx)
}

func (c *ctxWithOnlyDb) SetDb(conn *gorm.DB, isTransactional bool) {
	c.db = conn
	c.isTransactional = isTransactional
}

func (c *ctxWithOnlyDb) GetDb() *gorm.DB {
	return c.db
}

// returns true, if #SetRollbackOnly() has been called
func (c *ctxWithOnlyDb) IsRollbackOnly() bool {
	return c.rollbackOnly
}

// sets up the transaction to be rolled back
func (c *ctxWithOnlyDb) SetRollbackOnly() {
	if c.db == nil {
		// ignore
	} else {
		if !c.isTransactional {
			panic("unable to rollback non-transactional database connection - call database.WithTx()")
		} else {
			c.rollbackOnly = true
		}
	}
}

func (c *ctxWithOnlyDb) GetGinCtx() *gin.Context {
	return nil
}

func (c *ctxWithOnlyDb) QueryParamAsBooleanWithDefault(name string, defaultValue bool) bool {
	panic("not supported")
}

// unknown and empty query params are returned as ""
func (c *ctxWithOnlyDb) QueryParamAsString(name string) string {
	panic("not supported")
}

func (c *ctxWithOnlyDb) QueryParamAsInt(name string) (int, error) {
	panic("not supported")
}

func (c *ctxWithOnlyDb) UnmarshalRequestBody(a any) error {
	panic("not supported")
}

func (c *ctxWithOnlyDb) GetUser() (*jwt.User, error) {
	return &jwt.User{Username: c.username, UserId: c.userId, Expires: 0, Roles: c.roles, UserContext: map[string]string{}}, nil
}

func (c *ctxWithOnlyDb) UserHasARole(rolesAllowed []string) (bool, error) {
	user, err := c.GetUser()
	return UserHasARole(rolesAllowed, user), err
}

func (c *ctxWithOnlyDb) StartSpan(name string, isRemote bool) trace.Span {
	tp := otel.GetTracerProvider()
	tracer := tp.Tracer(name)
	kind := trace.SpanKindInternal
	if isRemote {
		kind = trace.SpanKindClient
	}
	_, span := tracer.Start(context.Background(), name, trace.WithSpanKind(kind))
	return span
}

func (c *ctxWithOnlyDb) Debug(format string, a ...any) {
	l := c.getLog()
	l.Debug().Msgf(format, a...)
}

func (c *ctxWithOnlyDb) Info(format string, a ...any) {
	l := c.getLog()
	l.Info().Msgf(format, a...)
}

func (c *ctxWithOnlyDb) Warn(format string, a ...any) {
	l := c.getLog()
	l.Warn().Msgf(format, a...)
}

func (c *ctxWithOnlyDb) Error(format string, a ...any) {
	l := c.getLog()
	l.Error().Msgf(format, a...)
}

func (c *ctxWithOnlyDb) HandleError(err error, msg string, log zerolog.Logger) {
	id := uuid.NewString()
	c.SetRollbackOnly()
	log.Warn().Msgf("%s %s: %+v", id, msg, err)
}

func (c *ctxWithOnlyDb) getLog() zerolog.Logger {
	packageName, _ /*funcName*/ := getCallerInfo(3)

	span := trace.SpanFromContext(context.Background())
	spanCtx := span.SpanContext()

	traceId := spanCtx.TraceID().String()
	// spanId := spanCtx.SpanID().String()

	return logging.GetLog(packageName).With().Str("traceId", "tid:"+traceId).Logger()
}

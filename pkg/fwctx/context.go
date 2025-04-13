package fwctx

// pass the context around. no such thing as a request scoped bean.
// so let's pimp it a little and make it nicer to use
// see https://www.reddit.com/r/golang/comments/1c03tz6/how_to_use_context_implicitly_in_go/

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/abstratium-informatique-sarl/stratis/pkg/jwt"
	"github.com/abstratium-informatique-sarl/stratis/pkg/logging"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

const TOKEN_COOKIE_NAME = "token"
const _USER_KEY = "user"

// a stateless wrapper around gin.Context providing useful methods like SetDb() and GetDb() in order to 
// access the DB connection, and IsRollbackOnly() and SetRollbackOnly() to control the transactional state,
// and GetUser() to get the user from the JWT token if it was sent to the call. Also contains helper methods
// to log debug, info, warn and error messages, as well as to check if the user has a role and to get query parameters.
type ICtx interface {
	SetDb(conn *gorm.DB, isTransactional bool)
	GetDb() *gorm.DB
	IsRollbackOnly() bool
	SetRollbackOnly()
	GetGinCtx() *gin.Context
	QueryParamAsBooleanWithDefault(string, bool) bool
	QueryParamAsString(string) string
	QueryParamAsInt(string) (int, error)
	UnmarshalRequestBody(o any) error
	GetUser() (*jwt.User, error)
	UserHasARole(rolesAllowed []string) (bool, error)
	Info(format string, a ...any)
	Debug(format string, a ...any)
	Warn(format string, a ...any)
	Error(format string, a ...any)
	
	// TODO why hand over a logger, rather than simply calling getLog()?
	HandleError(err error, msg string, log zerolog.Logger)
	StartSpan(name string, isRemote bool) trace.Span
}

var highestId int = 0

var ErrorTokenNotFound = errors.New("STRATIS-1002 no valid matching token found")
var ErrorTokenWrong = errors.New("STRATIS-1001 token and hash do not match")

type ctx struct {
	id int
	ginCtx *gin.Context
	
	// a function that provides the context if any, userId, userName, roles, or an error.
	contextProvider func(ICtx, string) (jwt.UserContext, string, string, []string, error)
}

func(c *ctx) setContextProviderIfNotSet(contextProvider func(ICtx, string) (userContext jwt.UserContext, userId string, userName string, roles []string, err error)) {
	if c.contextProvider == nil {
		c.contextProvider = contextProvider // reset the context in case one has now been provided
	}
}

func (c *ctx) SetDb(conn *gorm.DB, isTransactional bool) {
	c.ginCtx.Set("DB_CONN", conn)
	c.ginCtx.Set("DB_CONN_IS_TRANSACTIONAL", isTransactional)
}

func (c *ctx) GetDb() *gorm.DB {
	conn, _ := c.ginCtx.Get("DB_CONN")
	if conn == nil {
		panic("use TxMiddleware or NonTxMiddleware to setup the database for this call")
	}
	return conn.(*gorm.DB)
}

// returns true, if #SetRollbackOnly() has been called
func (c *ctx) IsRollbackOnly() bool {
	return c.ginCtx.GetBool("IS_ROLLBACK_ONLY")
}

// sets up the transaction to be rolled back
func (c *ctx) SetRollbackOnly() {
	if _, ok := c.ginCtx.Get("DB_CONN"); !ok {
		// ignore
	} else {
		if !c.ginCtx.GetBool("DB_CONN_IS_TRANSACTIONAL") {
			panic("unable to rollback non-transactional database connection - use TxMiddleware() for this call or call database.WithTx() if not in the context of an http call")
		} else {
			c.ginCtx.Set("IS_ROLLBACK_ONLY", true)
		}
	}
}

func (c *ctx) GetGinCtx() *gin.Context {
	return c.ginCtx
}

func (c *ctx) QueryParamAsBooleanWithDefault(name string, defaultValue bool) bool {
	b, err := strconv.ParseBool(c.ginCtx.DefaultQuery(name, fmt.Sprintf("%v", defaultValue)))
	if err != nil { return false }
	return b
}

// unknown and empty query params are returned as ""
func (c *ctx) QueryParamAsString(name string) string {
	return c.ginCtx.Query(name)
}

func (c *ctx) QueryParamAsInt(name string) (int, error) {
	i, e := strconv.ParseInt(c.ginCtx.Query(name), 10, 32)
	return int(i), e
}

func (c *ctx) UnmarshalRequestBody(a any) error {
	defer c.ginCtx.Request.Body.Close()
    reqBody, err := io.ReadAll(c.ginCtx.Request.Body)
	if err == nil {
		log := logging.GetLog("ctx")
		log.Debug().Msgf("got body: %+v", string(reqBody))
		err = json.Unmarshal(reqBody, a)
	}
	return err
}

func (c *ctx) GetUser() (*jwt.User, error) {
	obj, ok := c.ginCtx.Get(_USER_KEY)
	if !ok || obj == nil {
		var token string
		for _, c := range c.ginCtx.Request.Cookies() {
			if c.Name == "token" {
				token = c.Value
				break
			}
		}

		var user *jwt.User
		var err error
		if len(token) == 0 {
			// perhaps it comes out of the Authorization header
			tokenList := c.ginCtx.Request.Header["Authorization"]
			if len(tokenList) == 0 || len(tokenList[0]) == 0 {
				// no, it isn't present either
				user = &jwt.User{Username: jwt.ANONYMOUS, UserId: "0", Expires: 0, Roles: []string{}, UserContext: map[string]string{}}
			} else {
				if c.contextProvider == nil {
					err = fmt.Errorf("STRATIS-1000 contextProvider is nil but must be set for calls coming from %s %s. This is a bug, please inform an administrator", c.ginCtx.Request.Method, c.ginCtx.Request.RequestURI)
				} else {
					var userContext jwt.UserContext
					var roles []string
					var userId string
					var userName string
					userContext, userId, userName, roles, err = c.contextProvider(c, tokenList[0])
					if err == nil {
						expires := float64(time.Now().Add(5*time.Minute).Unix())
						user = &jwt.User{Username: userName, UserId: userId, Expires: expires, Roles: roles, UserContext: userContext}
					}
				}
			}
		} else {
			user, err = jwt.VerifyToken(token)
		}
		if err == nil {
			c.ginCtx.Set(_USER_KEY, user)
		}
		return user, err
	} else {
		return obj.(*jwt.User), nil
	}
}

func (c *ctx) UserHasARole(rolesAllowed []string) (bool, error) {
	user, err := c.GetUser()
	if err != nil {
		return false, err
	}
	return UserHasARole(rolesAllowed, user), nil
}

func (c *ctx) StartSpan(name string, isRemote bool) trace.Span {
	tp := otel.GetTracerProvider()
	tracer := tp.Tracer(name)
	kind := trace.SpanKindInternal
	if isRemote {
		kind = trace.SpanKindClient
	}
	_, span := tracer.Start(c.ginCtx.Request.Context(), name, trace.WithSpanKind(kind))
	return span
}

func UserHasARole(rolesAllowed []string, user *jwt.User) bool {
	for _, role := range user.Roles {
		if lo.Contains(rolesAllowed, role) {
			return true
		}
	}
	return false
}

func BuildTypedCtx(c *gin.Context, contextProvider func(ICtx, string) (jwt.UserContext, string, string, []string, error)) ICtx {
	// reuse the wrapper if it is already in the gin context
	key := "fwctx"
	a, exists := c.Get(key)
	var context *ctx
	if exists && a != nil {
		context = a.(*ctx)
		context.setContextProviderIfNotSet(contextProvider)
	} else {
		// is this thread safe? well... id is only fo debugging
		id := highestId
		highestId++

		context = &ctx{id, c, contextProvider}
		c.Set(key, context)
	}
	return ICtx(context)
}

func (c *ctx) Debug(format string, a ...any) {
	l := c.getLog()
	l.Debug().Msgf(format, a...)
}

func (c *ctx) Info(format string, a ...any) {
	l := c.getLog()
	l.Info().Msgf(format, a...)
}

func (c *ctx) Warn(format string, a ...any) {
	l := c.getLog()
	l.Warn().Msgf(format, a...)
}

func (c *ctx) Error(format string, a ...any) {
	l := c.getLog()
	l.Error().Msgf(format, a...)
}

func (c *ctx) HandleError(err error, msg string, log zerolog.Logger) {
	id := uuid.NewString()
	resp := gin.H{
		"id": id,
		"error": fmt.Sprintf("%+v", err),
	}
	c.SetRollbackOnly()
	log.Warn().Msgf("%s %s: %+v", id, msg, err)
	c.ginCtx.JSON(http.StatusBadRequest, resp)
}


func (c *ctx) getLog() zerolog.Logger {
	packageName, _/*funcName*/ := getCallerInfo(3)

	span := trace.SpanFromContext(c.ginCtx.Request.Context())
    spanCtx := span.SpanContext()

	traceId := spanCtx.TraceID().String()
	// spanId := spanCtx.SpanID().String()

	return logging.GetLog(packageName).With().Str("traceId", "tid:" + traceId).Logger()
}

func getCallerInfo(skip int) (packageName, funcName string) {
	pc, file, _, ok := runtime.Caller(skip)
	if !ok {
		return "", "" 
	}

	funcName = runtime.FuncForPC(pc).Name()
	lastSlash := strings.LastIndexByte(funcName, '/')
	if lastSlash < 0 {
		lastSlash = 0 
	}
	lastDot := strings.LastIndexByte(funcName[lastSlash:], '.') + lastSlash

	packageName = funcName[:lastDot]

	funcName = funcName[lastDot+1:]

	pathParts := strings.Split(file, "/")
	l := len(pathParts)
	file = "unknown"
	if l > 0 {
		file = pathParts[l-1]
	}

	// remove trailing ".go" if present
	i := strings.Index(file, ".go")
	if i > 0 {
		file = file[0:i]
	}

	return packageName + "/" + file, funcName
}	

package framework_gin

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/abstratium-informatique-sarl/stratis/pkg/database"
	"github.com/abstratium-informatique-sarl/stratis/pkg/fwctx"
	"github.com/abstratium-informatique-sarl/stratis/pkg/jwt"
	"github.com/abstratium-informatique-sarl/stratis/pkg/logging"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// https://github.com/prometheus/client_golang/blob/main/examples/exemplars/main.go
var opsProcessed *prometheus.CounterVec
var opsHistogramProcessed *prometheus.HistogramVec
var opsSummaryProcessed *prometheus.SummaryVec

func Setup(prefix string) {
    // https://github.com/prometheus/client_golang/blob/main/examples/exemplars/main.go
    opsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: prefix + "_response_count",
        Help: "The total number of calls processed, regardless of status code",
    },[]string{"code", "full_path_with_method"})

    opsHistogramProcessed = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name: prefix + "_response_latency_histogram",
        Help: "The response latency in ms of successful calls",
        Buckets: prometheus.ExponentialBuckets(4, 2, 6),
    },[]string{"full_path_with_method"})

    opsSummaryProcessed = promauto.NewSummaryVec(prometheus.SummaryOpts{
        Name: prefix + "_response_latency_summary",
        Help: "The response latency in ms of successful calls",
    },[]string{"full_path_with_method"})
}

// ================================================================================================
// timing middleware - more complex, so that it has access to writing headers into the response
// ================================================================================================

// https://github.com/gin-gonic/gin/issues/2406#issuecomment-1485704921
type timingMiddlewareWriter struct {
    gin.ResponseWriter
    start time.Time
    ctx fwctx.ICtx
}

func (w *timingMiddlewareWriter) WriteHeader(statusCode int) {
    if statusCode > 0 {
        elapsed := time.Since(w.start)
        w.ctx.Debug("timer ended after writing statusCode " + fmt.Sprintf("%d", statusCode) + ": " + elapsed.String())
        w.Header().Add("x-time", fmt.Sprintf("%v", elapsed.Milliseconds()))

        fp := w.ctx.GetGinCtx().FullPath()
        fpwm := w.ctx.GetGinCtx().Request.Method + " " + fp
        code := fmt.Sprintf("%d", w.Status())

        opsProcessed.With(prometheus.Labels{"code": code, "full_path_with_method": fpwm}).Inc()

        if statusCode >= 200 && statusCode < 300 {
            opsHistogramProcessed.With(prometheus.Labels{"full_path_with_method": fpwm}).Observe(float64(elapsed.Milliseconds()))
            opsSummaryProcessed.With(prometheus.Labels{"full_path_with_method": fpwm}).Observe(float64(elapsed.Milliseconds()))
        }

        // fetch the span again, in case it has now changed
        span := trace.SpanFromContext(w.ctx.GetGinCtx().Request.Context())

        user, err := w.ctx.GetUser()
        if err != nil && user != nil {
            span.SetAttributes(attribute.String("u", user.UserId))
        } // else ignore the fact that we cannot get the user
        spanCtx := span.SpanContext()

        traceId := spanCtx.TraceID().String()
        w.Header().Add("x-trace-id", traceId)
        w.ResponseWriter.WriteHeader(statusCode)
    } // else sometimes the framework calls this when the status isn't actually set to a proper number
}

func TimingMiddleware(c *gin.Context) {
    ctx := fwctx.BuildTypedCtx(c, nil)
    ctx.Debug("=========================")
    ctx.Debug("timer starting '" + c.Request.Method + " " + c.Request.URL.String() + "'...")

    c.Writer = &timingMiddlewareWriter{ c.Writer, time.Now(), ctx}
    c.Next()
    ctx.Debug("timer ended after next()") // too late to add headers here, if the handler has already written the status code
}

// =========================================================================================================================
// normal middleware that encapsulates a transaction (cannot write headers after call to .next() -> see timing middleware
// =========================================================================================================================
func TxMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := fwctx.BuildTypedCtx(c, nil)

        // ignore return, since we return nil,nil in 3 lines time
        _, err := database.WithTx(ctx, func() (any, error) {
            c.Next()
            return nil, nil
        })
        if err != nil {
            log := logging.GetLog("tx-middleware")
            log.Error().Msgf("failed to run transaction %+v", err)
            // would like to AbortWithError, but I don't know if that actually works, 
            // if the user code already wrote the header (status code). and it is supposed to do that!
            // the chances are high that we are failing during commit, which happens after user code is 
            // run.
            //         c.AbortWithError(http.StatusInternalServerError, err)
            // so, we just panic, and let gin handle it
            panic(err)
        }
    }
}

// ================================================================================================================================
// normal middleware that encapsulates a non-transaction call, but still puts the DB into the context, just without a transaction
// ================================================================================================================================
func NonTxMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        ctx := fwctx.BuildTypedCtx(c, nil)

        database.NonTx(ctx)
        
        c.Next()
    }
}

// ================================================================================================================================
// security middleware that ensures the user in the ICtx has one of the required roles which ultimately comes out of 
// a JWT that has been verified. An optional contextProvider can be provided to handle service users.
// ================================================================================================================================
func SecurityMiddleware(rolesAllowed []string, contextProvider func(fwctx.ICtx, string) (jwt.UserContext, string, string, []string, error)) gin.HandlerFunc {
    secLog := logging.GetLog("sec-middleware")
    return func(c *gin.Context) {
        ctx := fwctx.BuildTypedCtx(c, contextProvider)
        ok, err := ctx.UserHasARole(rolesAllowed)
        if err != nil {
            if errors.Is(err, fwctx.ErrorTokenNotFound) || errors.Is(err, fwctx.ErrorTokenWrong) {
                c.AbortWithError(http.StatusUnauthorized, err)
            } else {
                secLog.Error().Msgf("failed to check roles %+v", err)
                c.AbortWithError(http.StatusInternalServerError, err)
            }
            return
        } else if ok {
            c.Next()
        } else { // role missing
            user, err := ctx.GetUser() // ignore error - it was non-nil when we just queried the roles, and is cached in the ctx
            if err != nil {
                secLog.Error().Msgf("failed to get user %+v", err)
                c.AbortWithError(http.StatusInternalServerError, err)
                return
            } else if user.IsAnonymous() {
                c.AbortWithStatus(http.StatusUnauthorized) // sign in
            } else {
                c.AbortWithStatus(http.StatusForbidden) // missing role
            }
        }
    }
}

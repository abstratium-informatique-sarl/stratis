package framework_gin

import (
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/abstratium-informatique-sarl/stratis/pkg/database"
	"github.com/gin-gonic/gin"
)

// ================================================================================================
// ping
// ================================================================================================
// https://quarkus.io/guides/smallrye-health
// https://www.reddit.com/r/kubernetes/comments/wayj42/what_should_readiness_liveness_probe_actually/
func AddPing(router *gin.Engine, buildNumber string) {
    startTime := time.Now()
    router.GET("/ping", func(c *gin.Context) {
        uptime := time.Since(startTime)
        hostname, _ := os.Hostname()

        resp := gin.H{
            "service":      "ok",
            "database":     "ok",
            "build-number": buildNumber,
            "live":         "ok",
            "ready":        "ok",
            "uptime":       uptime.String(),
        }

        /*
Metrics:
Request Count: Number of requests processed since startup.
Error Count: Number of errors encountered.
Latency: Average response time for requests.
Cache Hits/Misses: If you use caching, provide statistics about cache performance.
Feature Flags: Status of any feature flags.
         */

        if c.Query("memory") == "true" {
            var mem runtime.MemStats
            runtime.ReadMemStats(&mem)
            resp["memory"] = mem
        }

        if c.Query("extra") == "true" {
            resp["gin-version"] =   gin.Version
            resp["go-version"] =    runtime.Version()
            resp["numGoRoutines"] = runtime.NumGoroutine()
            resp["numCPU"] =      runtime.NumCPU()
            resp["GOOS"] =        runtime.GOOS
            resp["GOARCH"] =      runtime.GOARCH
            resp["GOROOT"] =      runtime.GOROOT()
            resp["hostname"] =    hostname

            env := map[string]string{}
            for _, element := range os.Environ() {
                variable := strings.Split(element, "=")
                if strings.Contains(strings.ToLower(variable[0]), "password") {
                    variable[1] = "***"
                } 
                env[variable[0]] = variable[1]
            }
            resp["env"] = env

            debug, _ := debug.ReadBuildInfo()
            resp["debug"] = debug
        }

        if err := database.Ping(); err != nil {
            resp["database"] = err.Error()
            resp["live"] = "nok"
        }

        c.JSON(http.StatusOK, resp)
    })
}


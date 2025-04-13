package web

import (
	"net/http"
	"os"
	"regexp"

	"github.com/abstratium-informatique-sarl/stratis/pkg/logging"
	"github.com/gin-gonic/gin"
)

var log = logging.GetLog("web")

// handle angular - where a user enters a route into the browser url, which cannot be found here,
// so we load the index page and let angular attempt to resolve the url
func AddNoRouteHandlerForWebFrameworks(router *gin.Engine) {
    webDir := "./web-dist" // handle dev (local)
    if _, err := os.Stat(webDir); (err != nil || os.IsNotExist(err)) {
        webDir = "/web-dist" // handle test/prod
    }
    fileNamePattern, err := regexp.Compile(`.*[.][a-zA-Z\d]+`)
    if err != nil { panic(err) }
    router.NoRoute(func(c *gin.Context) {
        log.Debug().Msgf("no route handling %s", c.Request.URL.Path)
        if c.Request.URL.Path == "/" {
            c.File(webDir + "/index.html")
        } else if fileNamePattern.MatchString(c.Request.URL.Path) {
            c.FileFromFS(c.Request.URL.Path, http.Dir(webDir))
        } else {
            // We could not find the resource, i.e. it is not anything known to the server (i.e. it is not a REST
            // endpoint or a servlet), and does not look like a file so try handling it in the front-end routes
            // and reset the response status code to 200.
            log.Debug().Msgf("turning 404 into 200 with index page so that angular can deal with it, for request to %s", c.Request.RequestURI)
            c.File(webDir + "/index.html")
        }
    })
    
}

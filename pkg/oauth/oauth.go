package oauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/abstratium-informatique-sarl/stratis/pkg/framework_gin"
	"github.com/abstratium-informatique-sarl/stratis/pkg/fwctx"
	"github.com/abstratium-informatique-sarl/stratis/pkg/jwt"
	"github.com/abstratium-informatique-sarl/stratis/pkg/logging"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"golang.org/x/crypto/bcrypt"
)

const _STRATIS_OAUTH_G_USERINFO_URL_ENV_NAME = "STRATIS_OAUTH_G_USERINFO_URL"
const _STRATIS_OAUTH_M_USERINFO_URL_ENV_NAME = "STRATIS_OAUTH_M_USERINFO_URL"
const _STRATIS_OAUTH_COOKIE_DOMAIN_NAME = "STRATIS_OAUTH_COOKIE_DOMAIN"

type State struct {
	StateId   string
	Verifier  string
	TargetUrl string
	Context   map[string]string
}

type GoogleUser struct {
	Email string `json:"email"`
}

type GithubUser struct {
	Login string `json:"login"`
}

type MicrosoftUser struct {
	Login string `json:"TODO"`
}

type SignInRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var stateCache map[string]*State = make(map[string]*State)

var _useGoogle bool
var _useMicrosoft bool
var _useOwn bool

func Setup(useGoogle bool, useMicrosoft bool, useOwn bool) {
	log := logging.GetLog("oauth")

	_useGoogle = useGoogle
	_useMicrosoft = useMicrosoft
	_useOwn = useOwn
	
	// ///////////////////////////////////////////////////////////////////////////
	// check we have all the env vars in order to do oauth
	//   - fail fast, rather than the fist time a user tries to sign in
	// ///////////////////////////////////////////////////////////////////////////

	getGoogleOAuthConfig()
	getMicrosoftOAuthConfig()

	getUseSecureCookies()

	if _useGoogle {
		gUserInfoUrl := os.Getenv(_STRATIS_OAUTH_G_USERINFO_URL_ENV_NAME)
		if len(gUserInfoUrl) == 0 {
			panic("please set env var for oauth google user info url")
		}
	}

	if _useMicrosoft {
		mUserInfoUrl := os.Getenv(_STRATIS_OAUTH_M_USERINFO_URL_ENV_NAME)
		if len(mUserInfoUrl) == 0 {
			panic("please set env var for oauth microsoft user info url")
		}
	}

	log.Debug().Msgf("=======================================")
	log.Debug().Msgf(" OAUTH ENV")
	log.Debug().Msgf(" ")
	if _useGoogle {
		log.Debug().Msgf("STRATIS_OAUTH_G_REDIRECT_URL=%s", os.Getenv("STRATIS_OAUTH_G_REDIRECT_URL"))
		log.Debug().Msgf("STRATIS_OAUTH_G_AUTH_URL=%s", os.Getenv("STRATIS_OAUTH_G_AUTH_URL"))
		log.Debug().Msgf("STRATIS_OAUTH_G_TOKEN_URL=%s", os.Getenv("STRATIS_OAUTH_G_TOKEN_URL"))
		log.Debug().Msgf("STRATIS_OAUTH_G_USERINFO_URL=%s", os.Getenv("STRATIS_OAUTH_G_USERINFO_URL"))
	}

	if _useMicrosoft {
		log.Debug().Msg("----")
		log.Debug().Msgf("STRATIS_OAUTH_M_REDIRECT_URL=%s", os.Getenv("STRATIS_OAUTH_M_REDIRECT_URL"))
		log.Debug().Msgf("STRATIS_OAUTH_M_AUTH_URL=%s", os.Getenv("STRATIS_OAUTH_M_AUTH_URL"))
		log.Debug().Msgf("STRATIS_OAUTH_M_TOKEN_URL=%s", os.Getenv("STRATIS_OAUTH_M_TOKEN_URL"))
		log.Debug().Msgf("STRATIS_OAUTH_M_USERINFO_URL=%s", os.Getenv("STRATIS_OAUTH_M_USERINFO_URL"))
	}
}

// AddAll adds all the oauth endpoints. 
// accountProvider is a function that is given a string containing the username, and returns:
// - an `Account` object
// - an `error` if something went wrong, which can also be `gorm.ErrRecordNotFound` if the user does not exist
func AddAll(router *gin.Engine, accountProvider func(fwctx.ICtx, string) (Account, error)) {
	api := router.Group("/oauth").
		Use(framework_gin.NonTxMiddleware())

	if _useGoogle {
		api.GET("/sign-in/g", getSignInGoogle)
		api.GET("/g/redirect", func(c *gin.Context) {
			getRedirect(c, "google", accountProvider)
		})
	}
	
	if _useMicrosoft {
		api.GET("/sign-in/m", getSignInMicrosoft)
		api.GET("/m/redirect", func(c *gin.Context) {
			getRedirect(c, "microsoft", accountProvider)
		})
	}

	if _useOwn {
		api.POST("/sign-in/o", func(c *gin.Context) {
			getSignInOwn(c, accountProvider)
		})
		api.POST("/o/redirect", func(c *gin.Context) { // post, since js fetch will do the same method as was used for the query that caused the redirect
			getRedirect(c, "own", accountProvider)
		})
	}

	api.GET("/user", GetUser)
	api.GET("/sign-out", getSignOut)
}

func getSignInOwn(c *gin.Context, accountProvider func(fwctx.ICtx, string) (Account, error)) {
	ctx := fwctx.BuildTypedCtx(c, nil)
	defer c.Request.Body.Close()
	requestBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		ctx.Error("Error reading sign-in request body: %v", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	o := &SignInRequest{}
	err = json.Unmarshal(requestBody, o)
	if err != nil {
		ctx.Error("Error unmarshalling sign-in request body: %v", err)
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	account, err := accountProvider(ctx, o.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Status(http.StatusBadRequest)
			return
		} else {
			// don't give the user too much info, like the fact that we cannot read the user, in case it helps them guess usernames
			c.String(http.StatusBadRequest, fmt.Sprintf("%v", err))
			return
		}
	}

	// https://gowebexamples.com/password-hashing/
	// https://stackoverflow.com/a/16896216/458370 => don't use pepper
	// the following uses salt under the hood, and each generated hash is different, e.g.
	//   $2a$10$bMwmWWn9kTZYc41QfYJ8x.dSHzsF.8nn5XzCXp8aTnbbDdr0cm/QqPASS
	//   =====================================================
	//   $2a$10$4RRj5Ca5vNRJFk8.k4UCme6YHIMAfdatU4125/65h/aw4SQA5mqgOPASS
	// so you can't build a dictionary
	err = bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(o.Password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			// StatusBadRequest and not 40x, so that user cannot try and guess usernames
			c.Status(http.StatusBadRequest)
			return
		} else {
			c.String(http.StatusBadRequest, fmt.Sprintf("%v", err))
			return
		}
	}

	state := &State{StateId: uuid.NewString(), Context: make(map[string]string)}
	state.Context["username"] = o.Username
	AddState(state)

	c.Redirect(http.StatusTemporaryRedirect, "/oauth/o/redirect?state="+state.StateId+"&code=4321") // code aint used for own login
}

// auto-cleared in 30 seconds
func AddState(state *State){
	stateCache[state.StateId] = state
	clearStateIn30Seconds(state.StateId)
}

func getSignInGoogle(c *gin.Context) {
	getSignIn(c, getGoogleOAuthConfig())
}

func getSignInMicrosoft(c *gin.Context) {
	getSignIn(c, getMicrosoftOAuthConfig())
}

// https://pkg.go.dev/golang.org/x/oauth2#example-Config
func getSignIn(c *gin.Context, config *oauth2.Config) {

	targetUrl := c.Query("targetUrl")
	if len(targetUrl) == 0 {
		targetUrl = "/"
	}

	// use PKCE to protect against CSRF attacks
	// https://www.ietf.org/archive/id/draft-ietf-oauth-security-topics-22.html#name-countermeasures-6
	verifier := oauth2.GenerateVerifier()

	// Redirect user to consent page to ask for permission
	// for the scopes specified above.
	state := &State{StateId: uuid.NewString(), Verifier: verifier, TargetUrl: targetUrl}
	AddState(state)

	url := config.AuthCodeURL(state.StateId, oauth2.AccessTypeOnline, oauth2.S256ChallengeOption(verifier))

	c.Redirect(http.StatusTemporaryRedirect, url)
}

func getRedirect(c *gin.Context, provider string, accountProvider func(fwctx.ICtx, string)(Account, error)) {
	ctx := fwctx.BuildTypedCtx(c, nil)
	// http://localhost:4200/oauth/g/redirect?
	// 	state=state
	//  &code=4%2F0Aasdfasdf-o2aQ7asdfasdf-DXocEasdfasdfkAUAs_f87asdfasdfadf
	//  &scope=email+openid+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fuserinfo.email
	//  &authuser=0
	//  &prompt=consent

	// ignore for now. e.g. Referer is missing if user has to reload
	// not a super protection, but cannot hurt
	/*
		referers, ok := c.Request.Header["Referer"]
		if !ok {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("STRATIS-1009, %+v", c.Request.Header))
			return
		}
		if len(referers) != 1 {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("STRATIS-1010 %s", referers))
			return
		}
		referer := referers[0]
		if referer != "https://accounts.google.com/" {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("STRATIS-1011 %s", referer))
			return
		}
	*/

	stateId := c.Query("state")
	if len(stateId) == 0 {
		// TODO need to allow the user to retry the sign in, e.g. if the server rebooted and lost the cache
		// alternatively move the cache to the db
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("STRATIS-1004"))
		return
	}

	state, ok := GetState(stateId)
	if !ok {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("STRATIS-1005 %s", stateId))
		return
	}
	verifier := state.Verifier

	// Use the authorization code that is pushed to the redirect
	// URL. Exchange will do the handshake to retrieve the
	// initial access token
	var config *oauth2.Config
	if provider == "google" {
		config = getGoogleOAuthConfig()
	} else if provider == "microsoft" {
		config = getMicrosoftOAuthConfig()
	} else if provider == "own" {
		config = nil
	} else {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("STRATIS-1006 %s", provider))
		return
	}

	code := c.Query("code")
	if len(code) == 0 {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("STRATIS-1007"))
		return
	}
	ctxForTokenExchange := context.Background()
	var tok *oauth2.Token
	if provider != "own" {
		var err error
		tok, err = config.Exchange(ctxForTokenExchange, code, oauth2.VerifierOption(verifier))
		if err != nil {
			ctx.Error("error in %s oauth token exchange: %+v", provider, err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		ctx.Debug("token from %s: expiresIn=%d, expiry=%+v, tokenType=%s", provider, tok.ExpiresIn, tok.Expiry, tok.TokenType)
	}

	// TODO use `tok.RefreshToken`

	username := "unknown"
	if provider == "google" && _useGoogle {
		client := config.Client(ctxForTokenExchange, tok)
		url := os.Getenv(_STRATIS_OAUTH_G_USERINFO_URL_ENV_NAME)
		userResponse, err := client.Get(url)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer userResponse.Body.Close()
		body, err := io.ReadAll(userResponse.Body)
		if err != nil {
			ctx.Error("Error reading user body: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		o := &GoogleUser{}
		err = json.Unmarshal(body, o)
		if err != nil {
			ctx.Error("Error unmarshalling google user: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		username = o.Email
	} else if provider == "microsoft" && _useMicrosoft {
		client := config.Client(ctxForTokenExchange, tok)
		userResponse, err := client.Get(os.Getenv(_STRATIS_OAUTH_M_USERINFO_URL_ENV_NAME))
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer userResponse.Body.Close()
		body, err := io.ReadAll(userResponse.Body)
		if err != nil {
			ctx.Error("Error reading user body: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		// fmt.Printf("%s", string(body))

		o := &MicrosoftUser{}
		err = json.Unmarshal(body, o)
		if err != nil {
			ctx.Error("Error unmarshalling microsoft user: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		username = o.Login + "#microsoft" // to make it unique across all providers
	} else if provider == "own" && _useOwn {
		username = state.Context["username"]
	} else {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("STRATIS-1008 %s, %v/%v/%v", provider, _useGoogle, _useMicrosoft, _useOwn))
		return
	}

	fwc := fwctx.BuildTypedCtx(c, nil) // use wrapper with nicer API
	account, err := accountProvider(fwc, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Redirect(http.StatusTemporaryRedirect, "/register/"+base64.StdEncoding.EncodeToString([]byte(username))) // use base64 and not urlEncoding, otherwise we end up comparing to the file regex and the ".com" makes it think it should try loading a file which results in a 404, rather than loading the page
			return
		} else {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
	if account.Provider != provider {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("signed in with non-%s account but username '%s' was found: %s", provider, username, account.Provider))
	}

	jwToken, err := jwt.CreateSignedToken(account.Id, account.Username, account.Roles)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// TODO use account to create token with roles

	// TODO compare our timeout, to maxAge of provider token, and use lower one?
	setCookie(c, fwctx.TOKEN_COOKIE_NAME, jwToken)

	c.Redirect(http.StatusTemporaryRedirect, state.TargetUrl)
}

func GetState(stateId string) (*State, bool) {
	state, ok := stateCache[stateId]
	return state, ok
}

func setCookie(c *gin.Context, name string, value string) {
	__setCookie(c, name, value, 60*60)
}

func unsetCookie(c *gin.Context, name string) {
	__setCookie(c, name, "", -1)
}

func __setCookie(c *gin.Context, name string, value string, maxAgeSeconds int) {
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Set-Cookie
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(name, value, maxAgeSeconds, "/", os.Getenv(_STRATIS_OAUTH_COOKIE_DOMAIN_NAME), getUseSecureCookies(), true)
}

func GetUser(c *gin.Context) {
	ctx := fwctx.BuildTypedCtx(c, nil)
	user, err := ctx.GetUser()
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func getSignOut(c *gin.Context) {
	// TODO sign out of provider too
	unsetCookie(c, fwctx.TOKEN_COOKIE_NAME)
	c.Redirect(http.StatusTemporaryRedirect, "/")
}

func getGoogleOAuthConfig() *oauth2.Config {
	clientId := os.Getenv("STRATIS_OAUTH_G_CLIENT_ID")
	clientSecret := os.Getenv("STRATIS_OAUTH_G_CLIENT_SECRET")
	redirectUrl := os.Getenv("STRATIS_OAUTH_G_REDIRECT_URL")
	authUrl := os.Getenv("STRATIS_OAUTH_G_AUTH_URL")
	tokenUrl := os.Getenv("STRATIS_OAUTH_G_TOKEN_URL")

	if _useGoogle {
		if len(clientId) == 0 {
			panic("please set env var for oauth google client id")
		} else if len(clientSecret) == 0 {
			panic("please set env var for oauth google client secret")
		} else if len(redirectUrl) == 0 {
			panic("please set env var for oauth google redirect url")
		} else if len(authUrl) == 0 {
			panic("please set env var for oauth google auth url")
		} else if len(tokenUrl) == 0 {
			panic("please set env var for oauth google token url")
		}
	}

	oauthConf := &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Scopes:       []string{"email"}, // https://developers.google.com/identity/protocols/oauth2/scopes#openid-connect
		RedirectURL:  redirectUrl,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authUrl,
			TokenURL: tokenUrl,
		},
	}

	return oauthConf
}

func getMicrosoftOAuthConfig() *oauth2.Config {
	clientId := os.Getenv("STRATIS_OAUTH_M_CLIENT_ID")
	clientSecret := os.Getenv("STRATIS_OAUTH_M_CLIENT_SECRET")
	redirectUrl := os.Getenv("STRATIS_OAUTH_M_REDIRECT_URL")
	authUrl := os.Getenv("STRATIS_OAUTH_M_AUTH_URL")
	tokenUrl := os.Getenv("STRATIS_OAUTH_M_TOKEN_URL")

	if _useMicrosoft {
		if len(clientId) == 0 {
			panic("please set env var for oauth microsoft client id")
		} else if len(clientSecret) == 0 {
			panic("please set env var for oauth microsoft client secret")
		} else if len(redirectUrl) == 0 {
			panic("please set env var for oauth microsoft redirect url")
		} else if len(authUrl) == 0 {
			panic("please set env var for oauth microsoft auth url")
		} else if len(tokenUrl) == 0 {
			panic("please set env var for oauth microsoft token url")
		}
	}

	// register it here: https://portal.azure.com/#view/Microsoft_AAD_RegisteredApps/CreateApplicationBlade/quickStartType~/null/isMSAApp~/false
	// selected "Accounts in any organizational directory (Any Microsoft Entra ID tenant - Multitenant) and personal Microsoft accounts (e.g. Skype, Xbox)" 
	// named it "xyz"
	// redirect uri is for "web"
	// you can view app registrations here: https://portal.azure.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade
	// BUT you might have to click "view all applications in the directory"
	// redirect urls are managed here: App registrations > xyz > Manage > Authentication (https://portal.azure.com/#view/Microsoft_AAD_RegisteredApps/ApplicationMenuBlade/~/Authentication/appId/<id>/isMSAApp~/false)
	// secrets are managed here: App registrations > xyz > Manage > Certificates & secrets > Client secrets (https://portal.azure.com/#view/Microsoft_AAD_RegisteredApps/ApplicationMenuBlade/~/Credentials/appId/<id>/isMSAApp~/false)
	// endpoint list is found at App registrations > xyz > Endpoints

	oauthConf := &oauth2.Config{
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Scopes:       []string{"user:email"}, // ??? TODO
		RedirectURL:  redirectUrl,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authUrl,
			TokenURL: tokenUrl,
		},
	}

	return oauthConf
}

// returns secureCookie based on env, default is true
func getUseSecureCookies() bool {
	secureCookieString := os.Getenv("STRATIS_USE_SECURE_COOKIES")

	if len(secureCookieString) == 0 {
		secureCookieString = "true"
	}
	secureCookie, err := strconv.ParseBool(secureCookieString)
	if err != nil {
		panic("please set env var for secure cookie to true or false")
	}

	return secureCookie
}

func clearStateIn30Seconds(stateId string) {
	time.AfterFunc(30*time.Second, func() {
		delete(stateCache, stateId)
	})
}

type Account struct {
	Id string
	Username string
	Provider string
	PasswordHash string
	Roles []string
}

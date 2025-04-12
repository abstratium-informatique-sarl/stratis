package jwt

import (
	"fmt"
	"os"
	"time"

	gljwt "github.com/golang-jwt/jwt/v5"
	"github.com/abstratium-informatique-sarl/stratis/pkg/logging"
	"github.com/rs/zerolog"
	"github.com/samber/lo"
)

const _STRATIS_JWT_KEY_ENV_NAME = "STRATIS_JWT_KEY"
const _STRATIS_JWT_ISSUER_ENV_NAME = "STRATIS_JWT_ISSUER"

const ANONYMOUS = "anonymous"

type UserContext map[string]string

type User struct {
	Username string `json:"username"`
	UserId string `json:"userid"`
	Expires float64 `json:"expires"`
	Roles []string `json:"roles"`
	UserContext UserContext `json:"usercontext"`
}

func (u *User) IsAnonymous() bool {
	return u.Username == ANONYMOUS
}

var log zerolog.Logger

func Setup() {
	log = logging.GetLog("jwt")

	jwtIssuer := os.Getenv(_STRATIS_JWT_ISSUER_ENV_NAME)
	jwtKey := os.Getenv(_STRATIS_JWT_KEY_ENV_NAME)
    
	if len(jwtIssuer) == 0 {
        panic("please set env var for jwt issuer")
	} else if len(jwtKey) == 0 {
        panic("please set env var for jwt key")
	}

	log.Debug().Msgf("=======================================")
	log.Debug().Msgf(" JWT ENV")
	log.Debug().Msgf(" ")
	log.Debug().Msgf("%s=%s", _STRATIS_JWT_ISSUER_ENV_NAME, os.Getenv(_STRATIS_JWT_ISSUER_ENV_NAME))
	log.Debug().Msgf("%s=<hidden>", _STRATIS_JWT_KEY_ENV_NAME)
}


// https://pkg.go.dev/github.com/golang-jwt/jwt/v5#example-New-Hmac
func CreateSignedToken(accountId string, accountUsername string, roles []string) (tokenString string, err error) {

	// mapping roles to strings
	// https://github.com/mariomac/gostream#example-3-generation-from-an-iterator-map-to-a-different-type
	// https://github.com/robpike/filter
	// https://stackoverflow.com/questions/49468242/idiomatic-replacement-for-map-reduce-filter-etc
	// 4.8k https://github.com/thoas/go-funk uses reflection
	// winner: 18k https://github.com/samber/lo which uses generics and performs better
	token := gljwt.NewWithClaims(gljwt.SigningMethodHS256, gljwt.MapClaims{
		"exp": time.Now().Add(60*time.Minute).Unix()*1_000, // ms
		"uid": accountId,
		"sub": accountUsername,
		"iss": os.Getenv(_STRATIS_JWT_ISSUER_ENV_NAME),
		"roles": roles,
	})

	// Sign and get the complete encoded token as a string using the secret
	return token.SignedString([]byte(os.Getenv(_STRATIS_JWT_KEY_ENV_NAME)))
}

// https://pkg.go.dev/github.com/golang-jwt/jwt/v5#example-Parse-Hmac
func VerifyToken(jwToken string) (*User, error) {
	token, err := gljwt.Parse(jwToken, func(token *gljwt.Token) (any, error) {
		if a, ok := token.Method.(*gljwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		} else if a.Name != "HS256" {
			return nil, fmt.Errorf("unexpected signing algo: %v", a.Name)
		}
	
		return []byte(os.Getenv(_STRATIS_JWT_KEY_ENV_NAME)), nil
	})
	if err != nil { return nil, err }

	t := token.Claims.(gljwt.MapClaims)
	iss, err := t.GetIssuer()
	if err != nil { return nil, err }

	if iss != os.Getenv(_STRATIS_JWT_ISSUER_ENV_NAME) {
		return nil, fmt.Errorf("unsupported issuer %s", iss)
	}

	username := t["sub"].(string)

	userid := t["uid"].(string)

	expiry := t["exp"].(float64)

	// extracting a string array from jwt.MapClaims is totally horrible:
	slice, ok := t["roles"].([]any)
	if !ok { return nil, fmt.Errorf("token does not contain roles as expected: %+v", t) }
	roles := lo.Map(slice, func(e any, i int) string {
		return fmt.Sprint(e)
	})

	user := &User{
		Username: username,
		UserId:   userid,
		Expires:  expiry,
		Roles:    roles,
		UserContext: map[string]string{}, // currently, we don't support adding any context from a jwt token, that is reserved for service users and their tokens where the context is the application and organisation
	}

	return user, nil
}


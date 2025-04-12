package env

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/abstratium-informatique-sarl/stratis/pkg/logging"
	"github.com/joho/godotenv"
)

const _PROD = "prod"
var env string

// Setup will load the environment variables from the .env file and the privateEnvFileLocation
// privateEnvFileLocation is the path to the private environment variables file
// e.g. it might live in your home folder, but basically anywhere outside of your repository, 
// so that the secrets are not made public
func Setup(privateEnvFileLocation string) {
	log := logging.GetLog("env")

	env = os.Getenv("STRATIS_ENV") // empty means prod

	wd, _ := os.Getwd()
	log.Info().Msgf("=======================================")
	log.Info().Msgf(" ENV %s", env)
	log.Info().Msgf(" ")
	log.Info().Msgf("searching for env file, pwd=%s", wd)

	// determine where the files are - when tests run, they run with the working dir lower than the root
	dots := "."
	for {
		filename := dots + "/.env"
		_, err := os.Stat(filename)
		if err != nil {
			if os.IsNotExist(err) {
				// file don't exist
				log.Info().Msgf("no env file found in %s/%s, going up a directory...", wd, dots)
				dots = dots + "/.."
			} else {
				panic(fmt.Sprintf("Error checking file '%s': %+v\n", filename, err))
			}
		} else {
			break
		}
	}

	f, _ := filepath.Abs(wd + "/" + dots)
	if len(env) > 0 {
		log.Info().Msgf("loading %s/.env.%s", f, env)
		godotenv.Load(dots + "/.env." + env)
	}
	log.Info().Msgf("loading (adding) %s/.env", f)
	var err error
	err = godotenv.Load(dots + "/.env") // It's important to note that it WILL NOT OVERRIDE an env variable that already exists - consider the .env file to set dev vars or sensible defaults.
	if err != nil {
		panic(fmt.Sprintf("Error loading file '%s/.env': %+v\n", f, err))
	}

	// finally add privateEnvFileLocation if present, as it holds secrets for when developing locally
	_, err = os.Stat(privateEnvFileLocation)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info().Msgf("skipping env file %s as it does not exist", privateEnvFileLocation)
		} else {
			panic(fmt.Sprintf("Error checking file '%s': %+v\n", privateEnvFileLocation, err))
		}
	} else {
		log.Info().Msgf("loading (adding) env file %s", privateEnvFileLocation)
		godotenv.Load(privateEnvFileLocation) // It's important to note that it WILL NOT OVERRIDE an env variable that already exists - consider the .env file to set dev vars or sensible defaults.
	}

	if len(env) == 0 {
		env = _PROD
	}
}

func Getenv() string {
	return env
}

func GetenvIsNotProd() bool {
    return !GetenvIsProd()
}

func GetenvIsProd() bool {
    return env == _PROD
}

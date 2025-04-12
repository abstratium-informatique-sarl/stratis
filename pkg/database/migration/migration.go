package migration

import (
	"os"

	"github.com/abstratium-informatique-sarl/stratis/pkg/database"
	"github.com/abstratium-informatique-sarl/stratis/pkg/logging"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog"

	sqldblogger "github.com/simukti/sqldb-logger"
	"github.com/simukti/sqldb-logger/logadapter/zerologadapter"
)

var log zerolog.Logger

func init() {
    log = logging.GetLog("migration")
}

func Migrate() {
    if os.Getenv(database.TICKETS_SKIP_DB) == "true" {
        log.Info().Msgf("Skipping DB migrations because %s is set to true", database.TICKETS_SKIP_DB)
        return
    }
    log.Info().Msg("Starting DB migrations...")

    db := database.GetRawDb()

    // https://github.com/golang-migrate/migrate/blob/master/database/mysql/README.md

    // https://pkg.go.dev/github.com/simukti/sqldb-logger#section-readme
    sqlLog := logging.GetLog("sql-migr")
    loggerAdapter := zerologadapter.New(sqlLog)
    db = sqldblogger.OpenDriver(database.GetDatabaseSourceName(), db.Driver(), loggerAdapter /*, using_default_options*/) // db is STILL *sql.DB

    log.Debug().Msg("Pinging DB...")
    err := db.Ping() // to check connectivity and DSN correctness
    if err != nil {
        log.Fatal().Msgf("Failed %+v", err)
    }
    log.Debug().Msg("Pinged DB")

    driver, err := mysql.WithInstance(db, &mysql.Config{})
    if err != nil {
        log.Fatal().Msgf("Failed %+v", err)
    }

    dir, err := os.Getwd()
    if err != nil {
        log.Fatal().Msgf("Error getting working directory: %+v", err)
    }

    // Print the working directory.
    log.Info().Msgf("Working directory: %s", dir)

    migLoc := os.Getenv("TICKETS_DB_MIGRATION_LOCATION")
    if len(migLoc) == 0 {
        log.Fatal().Msg("Please set the migration location env var")
    }
    sourceURL := "file://" + migLoc
    dbName := database.GetDatabaseConfig().DBName
    log.Debug().Msgf("migrations url %s and dbName %s", sourceURL, dbName)

    m, err := migrate.NewWithDatabaseInstance(
        sourceURL,
        dbName,
        driver,
    )
    if err != nil {
        log.Fatal().Msgf("DB migrations failed %+v", err)
    }

    m.Log = &myLogger{verbose: true, log: log} // provide a log impl so that we get details about the migration
    err = m.Up() // do as many as possible

    if err != nil {
        if err.Error() == "no change" {
            version, dirty, err := m.Version()
            log.Info().Msgf("No DB migrations necessary. Version is %d, dirty %t, error: %+v", version, dirty, err)
        } else {
            log.Fatal().Msgf("DB migrations failed: %+v", err)
        }
    } else {
        version, dirty, err := m.Version()
        log.Info().Msgf("DB migrations completed. Version is %d, dirty %t, error: %+v", version, dirty, err)
    }

    log.Info().Msgf("DB Stats: %+v", db.Stats())
}

// myLogger struct implements the Logger interface
type myLogger struct {
    verbose bool
    log     zerolog.Logger
}

// Printf method for myLogger
func (l *myLogger) Printf(format string, v ...any) {
    if l.verbose {
        l.log.Debug().Msgf(format, v...)
    } else {
        l.log.Info().Msgf(format, v...)
    }
}

// Verbose method for myLogger
func (l *myLogger) Verbose() bool {
    return l.verbose
}

package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/abstratium-informatique-sarl/stratis/pkg/fwctx"
	"github.com/abstratium-informatique-sarl/stratis/pkg/logging"
	"go.opentelemetry.io/otel"
	gorm_mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlog "gorm.io/gorm/logger"

	"github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog"
	sqldblogger "github.com/simukti/sqldb-logger"
	"github.com/simukti/sqldb-logger/logadapter/zerologadapter"
	"gorm.io/plugin/opentelemetry/tracing"

	"gorm.io/plugin/prometheus"
)

const STRATIS_SKIP_DB = "STRATIS_SKIP_DB"

var rawDb *sql.DB
var db *gorm.DB
var log = logging.GetLog("database")

// a function that encapsulates the given function inside a transaction
//
// `f` a function to be called within the tx
func WithTx(ctx fwctx.ICtx, f func() (any, error)) (any, error) {
    // https://gorm.io/docs/transactions.html#A-Specific-Example

    // Note the use of tx as the database handle once you are within a transaction
    tx := Begin()

    err := tx.Error
    if err != nil {
        return nil, err
    }

    defer func() {
        // rollback if there was a panic
        if r := recover(); r != nil {
            if tx != nil {
                err = Rollback(tx)
                if err != nil {
                    log.Error().Msgf("unable to rollback transaction while handling an error: %+v", err)
                }
            }
            panic(r) // propagate the error up to gin, so that it can turn it into a failure
        }
    }()

    ctx.SetDb(tx, true)

    // propagate otel context from request to statement, so that when db is called, it results in child spans.
    // I found this: https://dev.to/vmihailenco/monitoring-gin-and-gorm-with-opentelemetry-53o0
    //   tx.WithContext(c.Request.Context())
    // But it doesn't work
    // so go with home brewed version, discovered by debugging:
    if ctx.GetGinCtx() != nil {
        tx.Statement.Context = ctx.GetGinCtx().Request.Context()
    }

    // call the callback. when called from TxMiddleware, result and err are both nil, because all it does is call c.Next()
    result, err := f()

    if ctx.IsRollbackOnly() {
        log.Info().Msgf("rolling back TX because ctx.IsRollbackOnly is marked as true")
        err2 := Rollback(tx)
        if err2 != nil {
            log.Error().Msgf("unable to rollback transaction %+v", err2)
        }
        return result, err
    } else if err != nil { // rollback if there was an error
        log.Info().Msgf("rolling back TX because of error: %+v", err)
        err2 := Rollback(tx)
        if err2 != nil {
            log.Error().Msgf("unable to rollback transaction %+v", err2)
        }
        return nil, err
    }

    // commit and return any error which occurs
    err = Commit(tx)
    if err != nil {
        log.Error().Msgf("unable to commit transaction %+v", err)
    } else {
        log.Debug().Msgf("committed transaction")
    }
    return result, err
}

// sets a non-transactional connection into the context
func NonTx(ctx fwctx.ICtx) {
    ctx.SetDb(db, false)
}

func GetDatabaseSourceName() string {
    // alternatively just build it as a string:
    // url := username + ":" + password + "@tcp(" + host + ":3306)/" + name + "?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true"
    return GetDatabaseConfig().FormatDSN()
}

func GetDatabaseConfig() *mysql.Config {
    username := os.Getenv("STRATIS_DB_USERNAME")
    password := os.Getenv("STRATIS_DB_PASSWORD")
    host := os.Getenv("STRATIS_DB_HOST")
    name := os.Getenv("STRATIS_DB_NAME")
    port := os.Getenv("STRATIS_DB_PORT")

    if len(username) == 0 {
        fmt.Printf("===== ENVS =====\n")
        for i, value := range os.Environ() {
            if strings.Contains(value, "STRATIS") {
                fmt.Printf("ENV %d) %s\n", i, value)
            }
        } 
        panic("please set env var for db username")
    } else if len(password) == 0 {
        panic("please set env var for db password")
    } else if len(host) == 0 {
        panic("please set env var for db host")
    } else if len(name) == 0 {
        panic("please set env var for db name")
    }

    if len(port) == 0 {
        port = "3306"
    }

    cfg := mysql.Config{
        User:   username,
        Passwd: password,
        Net:    "tcp",
        Addr:   host + ":" + port,
        DBName: name,
        Params: map[string]string{
            "charset": "utf8mb4",
            "parseTime": "True",
            "loc": "Local",
            "multiStatements": "true",
        },
    }

    return &cfg
}

func Ping() error {
    if db == nil { 
        // it hasn't been set up on purpose
        return nil 
    } else {
        sql, err := db.DB()
        if err != nil { return err }
        return sql.Ping()
    } 
}

func GetRawDb() *sql.DB {
    return rawDb
}

func setupRawDb() bool {
    log := logging.GetLog("database")
    log.Debug().Msgf("=======================================")
    log.Debug().Msgf(" DB ENV")
    log.Debug().Msgf(" ")

    if os.Getenv(STRATIS_SKIP_DB) == "true" {
        log.Debug().Msgf("Skipping DB setup because STRATIS_SKIP_DB is set to true")
        log.Debug().Msgf("=======================================")
        return false
    } else {
        dsn := GetDatabaseSourceName()
        log.Debug().Msg("DB Url is " + strings.Split(dsn, ":")[0] + ":***@" + strings.Split(dsn, "@")[1])
        log.Debug().Msgf("=======================================")
    
        // https://go.dev/doc/tutorial/database-access
        rawDb2, err := sql.Open("mysql", dsn) // db is a database handle representing a pool of zero or more underlying connections. It's safe for concurrent use by multiple goroutines. (https://pkg.go.dev/database/sql#DB)
        if err != nil { log.Fatal().Msgf("Failed %+v", err) }
    
        rawDb = rawDb2 // set the variable used publicly
        return true
    }
}

func GetDb() *gorm.DB {
    return db
}

var txCounter = 0
func Begin() *gorm.DB {
    txCounter++
    log.Debug().Msgf("\n\nXTX >>>>> Begin TX %d\n\n", txCounter)
    return db.Begin()
}

func Commit(tx *gorm.DB) error {
    log.Debug().Msgf("\n\nXTX >>>>> Commit TX %d\n\n", txCounter)
    txCounter--
    return tx.Commit().Error
}

func Rollback(tx *gorm.DB) error {
    log.Debug().Msgf("\n\nXTX >>>>> Rollback TX %d\n\n", txCounter)
    txCounter--
    return tx.Rollback().Error
}

func SetupDb() {
    if setupRawDb() {
        // https://pkg.go.dev/github.com/simukti/sqldb-logger#section-readme
        sqlLog := logging.GetLog("sql-repo")
        loggerAdapter := zerologadapter.New(sqlLog)
        dbWithLog := sqldblogger.OpenDriver(GetDatabaseSourceName(), rawDb.Driver(), loggerAdapter /*, using_default_options*/) // db is STILL *sql.DB

        log := logging.GetLog("gorm")
        gormLogger := gormlog.New(
            &myLogger{log: log},
            gormlog.Config{
            SlowThreshold:              3*time.Millisecond,   // Slow SQL threshold
            LogLevel:                   gormlog.Info,  // Log level
            IgnoreRecordNotFoundError: true,           // Ignore ErrRecordNotFound error for logger
            ParameterizedQueries:      true,           // Don't include params in the SQL log
            Colorful:                  true,          // Disable color
            },
        )
        
        config := gorm.Config{
            Logger: gormLogger,             // use the logger configured above
            SkipDefaultTransaction: true,   // enable starting our own transactions
            TranslateError: true,           // enable translating mysql codes into errors like gorm.ErrDuplicatedKey
        }

        db2, err := gorm.Open(gorm_mysql.New(gorm_mysql.Config{
            Conn: dbWithLog,
        }), &config)

        if err != nil {
            panic("failed to connect database")
        }

        // add metrics - https://gorm.io/docs/prometheus.html
        cfg := GetDatabaseConfig()
        err = db2.Use(prometheus.New(prometheus.Config{
            DBName:          cfg.Addr+"/"+cfg.DBName, // use `DBName` as metrics label
            // RefreshInterval: 15,    // Refresh metrics interval (default 15 seconds)
            // PushAddr:    use scraping instead    "prometheus pusher address", // push metrics if `PushAddr` configured
            StartServer:     false,  // we have it running already
            // HTTPServerPort:  8080,  // configure http server port, default port 8080 (if you have configured multiple instances, only the first `HTTPServerPort` will be used to start server)
            MetricsCollector: []prometheus.MetricsCollector {
            &prometheus.MySQL{
                VariableNames: []string{"Threads_running"}, // taken from "show status", or leave blank to get all
            },
            },  // user defined metrics
        }))
        if err != nil {
            panic(err)
        }

        // add telemetry
        // e.g. attr := attribute.KeyValue{Key: "mykey", Value: attribute.BoolValue(true)}
        if err := db2.Use(tracing.NewPlugin(/*tracing.WithAttributes(attr), */tracing.WithTracerProvider(otel.GetTracerProvider()))); err != nil {
            panic(err)
        }

        db = db2 // set the variable used publicly

        // TODO Configure connection pooling:
        // sqlDB.SetMaxIdleConns(5)
        // sqlDB.SetMaxOpenConns(10)
        // sqlDB.SetConnMaxLifetime(30 * time.Minute)
    }
}

type myLogger struct {
    log zerolog.Logger
}

// Printf method for myLogger
func (l *myLogger) Printf(format string, v ...any) {
    l.log.Debug().Msgf(format, v...)
}

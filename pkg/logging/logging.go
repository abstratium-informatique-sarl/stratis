package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

var RootLogger zerolog.Logger
var lvls map[string]string
var loggers map[string]zerolog.Logger

func readLevels() map[string]string {
	data, err := os.ReadFile("./log-config.json") // prod and debugging in vscode
	if err != nil {
		data, err = os.ReadFile("../log-config.json") // running from cmd line with air
		if err != nil {
			data, err = os.ReadFile("../../log-config.json") // tests?
			if err != nil {
				data, err = os.ReadFile("../../../log-config.json") // deeper tests?
				if err != nil {
					data, err = os.ReadFile("../../../../log-config.json") // even deeper tests?
					if err != nil {
						data, err = os.ReadFile("/root/log-config.json") // prod, since the first attempt failed
						if err != nil {
							panic(fmt.Errorf("failed to read log config: %w", err))
						}
					}
				}
			}
		}
	}

	// Unmarshal the JSON data into a map[string]string
	config := make(map[string]string)
	err = json.Unmarshal(data, &config)
	if err != nil {
		panic(fmt.Errorf("failed to read log config: %w", err))
	}

	fmt.Println("logging config is " + fmt.Sprintf("%s", config))

	return config
}

func setupLog() {
	if(lvls == nil) { 
		lvls = readLevels()
	
		// https://github.com/rs/zerolog
		zerolog.TimeFieldFormat = time.RFC3339Nano
		output := zerolog.ConsoleWriter{
			Out: os.Stdout, 
			TimeFormat: "2006-01-02T15:04:05.000",
			PartsOrder: []string{"time", "level", "component", "traceId", "message"},
			PartsExclude: []string{},
			FieldsExclude: []string{"component", "traceId"},
		}
		output.FormatLevel = func(i any) string {
			if i == nil {
				return "|GIN   |"
			}
			s := strings.ToUpper(fmt.Sprintf("%s", i))

			color := COLOR_NONE
			if(s == "WARN") {
				color = COLOR_RED
			} else if(s == "ERROR" || s == "FATAL" || s == "PANIC") {
				color = COLOR_LIGHT_RED
			}
			s = strings.ToUpper(fmt.Sprintf("%-6s", s))
			return "|" + color + s + COLOR_NONE + "|"
		}
		output.FormatMessage = func(i any) string {
			return fmt.Sprintf("| %s ", i)
		}
		output.FormatFieldName = func(i any) string {
			return fmt.Sprintf("%s:", i)
		}
		output.FormatFieldValue = func(i any) string {
			if i == nil {
				return "| -"
			} else {
				s := fmt.Sprintf("%s", i)
				if strings.HasPrefix(s, "pkg:") {
					return abbreviateIfNecessary(s[4:])
				} else if strings.HasPrefix(s, "tid:") {
					return fmt.Sprintf("|%straceId: %s%s", COLOR_CYAN, s[4:], COLOR_NONE)
				} else {
					return strings.ToUpper(s)
				}
			}
		}
		
		RootLogger = zerolog.New(output).With().Timestamp().Str("component", "pkg:root").Logger()
	}
}

// TODO use this instead: https://pkg.go.dev/github.com/agrison/go-commons-lang@v0.0.0-20240106075236-2e001e6401ef/stringUtils#Abbreviate
const MAX_LENGTH = 20
func abbreviateIfNecessary(s string) string {
	if len(s) == MAX_LENGTH {
		return s
	} else if len(s) < MAX_LENGTH {
		return s + strings.Repeat(" ", MAX_LENGTH-len(s)) // Right pad with spaces
	} else {
		return s[:8] + ".." // Abbreviate with ".."
	}
}

func GetLog(component string) zerolog.Logger {
	component = shortenString(component)
	if(lvls == nil) {
		setupLog()
	}
	l, ok := loggers[component]
	if !ok {
		level, ok := lvls[component]
		if !ok {
			level = lvls["root"]
		}
		l = RootLogger.With().Str("component", "pkg:" + component).Logger()
		lvl, err := zerolog.ParseLevel(level)
		if err != nil { panic(fmt.Sprintf("unknown level '%s' in log config", level)) }
		l = l.Level(lvl)
	}
	return l
}

func shortenString(fullPackage string) string {
	parts := strings.Split(fullPackage, "/")
	shortenedParts := make([]string, 0, len(parts))
	for i, part := range parts {
		if i < len(parts)-1 {
			shortenedParts = append(shortenedParts, string(part[0]))
		} else {
			shortenedParts = append(shortenedParts, part)
		}
	}
	return strings.Join(shortenedParts, ".")
}

// https://unix.stackexchange.com/a/174/206459
const (
    COLOR_NONE="\033[0m"
    COLOR_BLACK="\033[0;30m"
    COLOR_GRAY="\033[1;30m"
    COLOR_RED="\033[0;31m"
    COLOR_LIGHT_RED="\033[1;31m"
    COLOR_GREEN="\033[0;32m"
    COLOR_LIGHT_GREEN="\033[1;32m"
    COLOR_BROWN="\033[0;33m"
    COLOR_YELLOW="\033[1;33m"
    COLOR_BLUE="\033[0;34m"
    COLOR_LIGHT_BLUE="\033[1;34m"
    COLOR_PURPLE="\033[0;35m"
    COLOR_LIGHT_PURPLE="\033[1;35m"
    COLOR_CYAN="\033[0;36m"
    COLOR_LIGHT_CYAN="\033[1;36m"
    COLOR_LIGHT_GRAY="\033[0;37m"
    COLOR_WHITE="\033[1;37m"
)
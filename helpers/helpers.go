package helpers

import (
	"flag"
	"fmt"
	"runtime"
	"strings"

	log "github.com/sirupsen/logrus"
)

//TraceData trace data struct
type TraceData struct {
	File string
	Line int
	Fn   string
}

//SetLogger sets logger settings
func SetLogger(logLevelVar string) {
	level, err := log.ParseLevel(logLevelVar)
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)

	log.SetReportCaller(true)
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.QuoteEmptyFields = true
	customFormatter.FullTimestamp = true
	customFormatter.CallerPrettyfier = func(f *runtime.Frame) (string, string) {
		repopath := strings.Split(f.File, "/")
		function := strings.Replace(f.Function, "security-json-import/", "", -1)
		return fmt.Sprintf("%s\t", function), fmt.Sprintf(" %s:%d\t", repopath[len(repopath)-1], f.Line)
	}

	log.SetFormatter(customFormatter)
	fmt.Println("Log level set at:", level)
}

//Check logger for errors
func Check(e error, panicCheck bool, logs string, trace TraceData) {
	if e != nil && panicCheck {
		log.Error(logs, " failed with error:", e, " ", trace.Fn, " on line:", trace.Line)
		panic(e)
	}
	if e != nil && !panicCheck {
		log.Warn(logs, " failed with error:", e, " ", trace.Fn, " on line:", trace.Line)
	}
}

//Trace get function data
func Trace() TraceData {
	var trace TraceData
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Warn("Failed to get function data")
		return trace
	}

	fn := runtime.FuncForPC(pc)
	trace.File = file
	trace.Line = line
	trace.Fn = fn.Name()
	return trace
}

//Flags struct
type Flags struct {
	WorkersVar, WorkerSleepVar, SkipGroupIndexVar, SkipUserIndexVar, SkipPermissionIndexVar, HTTPSleepSecondsVar, HTTPRetryMaxVar, NumReposVar         int
	UsernameVar, ApikeyVar, URLVar, RepoVar, LogLevelVar, CredsFileVar, UserEmailDomainVar, UserGroupAssocationFileVar, SecurityJSONFileVar, PrefixVar string
	SkipUserImportVar, SkipGroupImportVar, SkipPermissionImportVar, UsersWithGroupsVar, UsersFromGroupsVar                                             bool
}

//SetFlags function
func SetFlags() Flags {
	var flags Flags
	//mandatory flags
	flag.StringVar(&flags.SecurityJSONFileVar, "securityJSONFile", "", "Security JSON file from Artifactory Support Bundle")
	flag.StringVar(&flags.UsernameVar, "user", "", "Username")
	flag.StringVar(&flags.ApikeyVar, "apikey", "", "API key or password")
	flag.StringVar(&flags.URLVar, "url", "", "Binary Manager URL")

	flag.StringVar(&flags.PrefixVar, "prefix", "import", "Optional prefix")
	flag.StringVar(&flags.CredsFileVar, "credsFile", "", "File with creds. If there is more than one, it will pick randomly per request. Use whitespace to separate out user and password")

	//config flags
	flag.StringVar(&flags.LogLevelVar, "log", "INFO", "Order of Severity: TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC")
	flag.IntVar(&flags.WorkersVar, "workers", 50, "Number of workers")
	flag.IntVar(&flags.WorkerSleepVar, "workerSleep", 5, "Worker sleep period in seconds")
	flag.IntVar(&flags.HTTPSleepSecondsVar, "httpSleep", 10, "HTTP request sleep period before a retry")
	flag.IntVar(&flags.HTTPRetryMaxVar, "retry", 5, "Retry attempt before failure")
	flag.IntVar(&flags.NumReposVar, "numRepos", 10, "Number of repos to randomly create")

	flag.Parse()
	return flags
}

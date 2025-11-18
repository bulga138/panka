package version

var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func GetVersion() string {
	return Version
}

func GetCommit() string {
	return Commit
}

func GetBuildTime() string {
	return BuildTime
}

func GetFullVersion() string {
	return Version + " (" + Commit + ") built at " + BuildTime
}

package config

var userAgent = ""

func GetUserAgent() string {
	return userAgent
}

func SetUserAgent(agent string) {
	userAgent = agent
}

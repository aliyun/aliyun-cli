package i18n

var language = "en"

func SetLanguage(lang string) {
	language = lang
}

func GetLanguage() string {
	return language
}

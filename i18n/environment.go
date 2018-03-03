package i18n

type LanguageCode string

const (
	Zh = LanguageCode("zh")
	En = LanguageCode("en")
)
var language = "en"

func SetLanguage(lang string) {
	language = lang
}

func GetLanguage() string {
	return language
}

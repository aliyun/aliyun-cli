package i18n

type Text struct {
	id string
	dic map[string]string
}

func (a *Text) Text() string {
	lang := GetLanguage()
	return a.Get(lang)
}

func (a *Text) Get(lang string) string {
	s, ok := a.dic[lang]
	if !ok {
		return ""
	}
	return s
}

func (a *Text) Put(lang string, txt string) {
	a.dic[lang] = txt
}


package i18n

var library = make(map[string]*Text)

func En(id string, text string) (*Text) {
	return putText(id, "en", text)
}

func Zh(id string, text string) (*Text) {
	return putText(id, "zh", text)
}

func T(en string, zh string) (*Text) {
	t := &Text {
		id: "",
		dic: make(map[string]string),
	}
	t.dic["en"] = en
	if zh != "" {
		t.dic["zh"] = zh
	}
	return t
}

func putText(id string, lang string, text string) (*Text) {
	t, ok := library[id]
	if !ok {
		t = &Text {
			id: id,
			dic: make(map[string]string),
		}
		library[id] = t
	}
	t.Put(lang, text)
	return t
}

func getText(id string, lang string) (string, bool) {
	t, ok := library[id]
	if !ok {
		return "", false
	}
	return t.Get(lang), ok
}
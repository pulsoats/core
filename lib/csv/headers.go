package csv

import (
	"reflect"
	"strings"
)

// CreateHeaders - создание заголовка по тегам/названиям полей в структуре
func CreateHeaders(v any) []string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	headers := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)

		// пропустить неэкспортируемые поля — их значение всё равно не прочитать
		if sf.PkgPath != "" {
			continue
		}

		tag := sf.Tag.Get("csv")
		name, _, _ := strings.Cut(tag, ",") // часть до запятой — имя, после — моды
		name = strings.TrimSpace(name)

		// пропустить поле, если имя — "-"
		if name == "-" {
			continue
		}

		// если имя в теге не задано — берём имя поля
		if name == "" {
			name = sf.Name
		}

		headers = append(headers, name)
	}
	return headers
}

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

		tag := sf.Tag.Get("csv")
		name, mods, _ := strings.Cut(tag, ",") // часть до запятой — имя, после — моды
		name = strings.TrimSpace(name)
		mods = strings.ReplaceAll(strings.TrimSpace(mods), " ", "") // " price, ppm" -> "price,ppm"

		// если тега нет — берём имя поля
		if name == "" {
			name = sf.Name
		}

		// пропустить поле, если указан модификатор omit
		if mods != "" {
			m := "," + mods + ","
			if strings.Contains(m, ",omit,") {
				continue
			}
		}

		headers = append(headers, name)
	}
	return headers
}

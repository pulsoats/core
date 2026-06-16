package csv

import (
	"reflect"
	"strings"
	"time"
)

var SignalHeader = []string{}

var timeType = reflect.TypeOf(time.Time{})

// CreateHeaders - создание заголовка по тегам/названиям полей в структуре.
// Анонимные встроенные структуры разворачиваются в отдельные колонки (как в encoding/json).
// Поле пропускается тегом csv:"-".
func CreateHeaders(v any) []string {
	t := reflect.TypeOf(v)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return headersOf(t)
}

func headersOf(t reflect.Type) []string {
	headers := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)

		// неэкспортируемые не-встроенные поля пропускаем
		if sf.PkgPath != "" && !sf.Anonymous {
			continue
		}

		name, _, _ := strings.Cut(sf.Tag.Get("csv"), ",") // часть до запятой — имя
		name = strings.TrimSpace(name)

		// пропустить поле
		if name == "-" {
			continue
		}

		// структура без явного имени → разворачиваем её поля
		if name == "" {
			ft := sf.Type
			for ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct && ft != timeType {
				headers = append(headers, headersOf(ft)...)
				continue
			}
		}

		// если имя в теге не задано — берём имя поля
		if name == "" {
			name = sf.Name
		}

		headers = append(headers, name)
	}
	return headers
}

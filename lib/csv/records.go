package csv

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/pulsoats/core/lib/format"
	"github.com/pulsoats/core/lib/units"
)

// StructToCSVRecord берет значения из структуры v согласно csv-тегам.
// Анонимные встроенные структуры разворачиваются в отдельные колонки (как в CreateHeaders).
// Поле пропускается тегом csv:"-".
// Поддерживаемые модификаторы (после запятой):
//   - price : int64 в центах -> деньги (CentsToString)
//   - ppm   : int64 в ppm    -> доля (0.xxxxxx)
//   - time  : unix sec/ms    -> "2006-01-02 15:04:05" (UTC)
//   - raw   : вывод как есть
func StructToCSVRecord(v any) []string {
	val := reflect.ValueOf(v)
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return recordOf(val)
}

// has проверяет наличие модификатора после запятой
// (устойчиво к формату "..., price,ppm").
func has(mods, key string) bool {
	if mods == "" {
		return false
	}
	m := "," + strings.ReplaceAll(mods, " ", "") + ","
	return strings.Contains(m, ","+key+",")
}

func recordOf(val reflect.Value) []string {
	typ := val.Type()
	values := make([]string, 0, typ.NumField())

	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		fv := val.Field(i)

		// неэкспортируемые не-встроенные поля пропускаем
		if sf.PkgPath != "" && !sf.Anonymous {
			continue
		}

		name, mods, _ := strings.Cut(sf.Tag.Get("csv"), ",")
		name = strings.TrimSpace(name)
		mods = strings.TrimSpace(mods)

		// пропустить поле
		if name == "-" {
			continue
		}

		// анонимная встроенная структура без явного имени → разворачиваем её поля
		if sf.Anonymous && name == "" {
			ft := sf.Type
			for ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct && ft != timeType {
				ev := fv
				for ev.Kind() == reflect.Ptr {
					if ev.IsNil() {
						// nil-указатель: добиваем пустыми ячейками под все колонки встроенной структуры
						values = append(values, make([]string, len(headersOf(ft)))...)
						ev = reflect.Value{}
						break
					}
					ev = ev.Elem()
				}
				if ev.IsValid() {
					values = append(values, recordOf(ev)...)
				}
				continue
			}
		}

		values = append(values, encodeField(fv, mods))
	}

	return values
}

// encodeField форматирует одно поле в строку согласно модификаторам тега.
func encodeField(fv reflect.Value, mods string) string {
	switch fv.Kind() {
	case reflect.String:
		return fv.String()

	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%.8f", fv.Float())

	case reflect.Int, reflect.Int32, reflect.Int64:
		vi := fv.Int()

		switch {
		case has(mods, "raw"):
			return fmt.Sprintf("%d", vi)

		case has(mods, "price"):
			// денежные поля в центах
			return format.CentsToString(vi)

		case has(mods, "ppm"):
			// ppm -> доля (0.xxxxxx)
			return fmt.Sprintf("%.6f", float64(vi)/float64(units.PPM))

		case has(mods, "time"):
			// unix seconds / millis -> "2006-01-02 15:04:05" (UTC)
			var tm time.Time
			if vi >= 1_000_000_000_000 {
				tm = time.UnixMilli(vi).UTC()
			} else {
				tm = time.Unix(vi, 0).UTC()
			}
			return tm.Format("2006-01-02 15:04:05")

		default:
			return fmt.Sprintf("%d", vi)
		}

	case reflect.Struct:
		// поддержка time.Time без модификатора
		if t, ok := fv.Interface().(time.Time); ok {
			return t.Format("2006-01-02 15:04:05")
		}
		return fmt.Sprint(fv.Interface())

	default:
		return fmt.Sprint(fv.Interface())
	}
}

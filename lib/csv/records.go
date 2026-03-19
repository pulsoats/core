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
// Поддерживаемые модификаторы (после запятой):
//   - price        : int64 в центах -> деньги (FormatCents)
//   - ppm          : int64 в ppm    -> доля (0.xxxxxx)
//   - time         : unix sec/ms    -> "2006-01-02 15:04:05" (UTC)
//   - raw          : вывод как есть
//   - omit         : пропустить поле
func StructToCSVRecord(v any) []string {
	val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)

	if typ.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	values := make([]string, 0, typ.NumField())

	// утилита проверки модификатора: принимает строку после запятой,
	// удаляет пробелы и ищет подстроку по запятым (устойчиво к формату "..., price,ppm")
	has := func(mods, key string) bool {
		if mods == "" {
			return false
		}
		m := "," + strings.ReplaceAll(mods, " ", "") + ","
		k := "," + key + ","
		return strings.Contains(m, k)
	}

	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		fv := val.Field(i)

		tag := sf.Tag.Get("csv")
		// name в этой функции не нужен, но из тега достаём mods (после запятой)
		_, mods, _ := strings.Cut(tag, ",")
		mods = strings.TrimSpace(mods)

		// omit: пропускаем поле полностью
		if has(mods, "omit") {
			continue
		}

		var s string

		switch fv.Kind() {
		case reflect.String:
			s = fv.String()

		case reflect.Float32, reflect.Float64:
			s = fmt.Sprintf("%.8f", fv.Float())

		case reflect.Int, reflect.Int32, reflect.Int64:
			vi := fv.Int()

			switch {
			case has(mods, "raw"):
				s = fmt.Sprintf("%d", vi)

			case has(mods, "price"):
				// денежные поля в центах
				s = format.FormatCents(vi)

			case has(mods, "ppm"):
				// ppm -> доля (0.xxxxxx)
				s = fmt.Sprintf("%.6f", float64(vi)/float64(units.PPM))

			case has(mods, "time"):
				// unix seconds / millis -> "2006-01-02 15:04:05" (UTC)
				var tm time.Time
				if vi >= 1_000_000_000_000 {
					tm = time.UnixMilli(vi).UTC()
				} else {
					tm = time.Unix(vi, 0).UTC()
				}
				s = tm.Format("2006-01-02 15:04:05")

			default:
				s = fmt.Sprintf("%d", vi)
			}

		case reflect.Struct:
			// поддержка time.Time без модификатора
			if t, ok := fv.Interface().(time.Time); ok {
				s = t.Format("2006-01-02 15:04:05")
			} else {
				s = fmt.Sprint(fv.Interface())
			}

		default:
			s = fmt.Sprint(fv.Interface())
		}

		values = append(values, s)
	}

	return values
}

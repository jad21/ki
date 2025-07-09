package templates

import (
	"encoding/json"
	"html/template"
	"regexp"
	"strings"
	"time"
)

// // FuncMap es un mapa de funciones para ser usadas en los templates.
// var FuncMap = map[string]any{
// 	"json":       toJSON,
// 	"caseSnake":  toSnakeCase,
// 	"min":        minFunc,
// 	"max":        maxFunc,
// 	"formatDate": formatDate,
// 	// No es necesario añadir 'eq', 'ne', 'lt', 'le', 'gt', 'ge', ya que son nativos.
// }

func FuncMapDefault() template.FuncMap {
	return template.FuncMap{
		"title":      strings.Title, // Obsoleto en Go 1.18+, pero funciona para demostración. Ver nota abajo.
		"json":       toJSON,
		"caseSnake":  toSnakeCase,
		"camelCase":  toCamelCase,
		"min":        minFunc,
		"max":        maxFunc,
		"formatDate": formatDate,
		"dic":        dict,
	}
}

// mergeFuncMaps combina varios FuncMap en uno solo.
// Las funciones en mapas posteriores sobrescriben a las anteriores si los nombres coinciden.
func mergeFuncMaps(maps ...template.FuncMap) template.FuncMap {
	merged := make(template.FuncMap)

	for _, m := range maps {
		for key, value := range m {
			merged[key] = value
		}
	}
	return merged
}

// toJSON convierte una estructura a una cadena JSON segura para JavaScript.
// Uso en template: {{ .MiStruct | json }}
func toJSON(v any) (template.JS, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return template.JS(b), nil
}

// toSnakeCase convierte un string de CamelCase a snake_case.
// Uso en template: {{ "MiVariable" | caseSnake }} -> "mi_variable"
var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// toCamelCase convierte un string de snake_case a UpperCamelCase (PascalCase).
// Ejemplo: "mi_variable_de_prueba" se convierte en "MiVariableDePrueba".
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(string(part[0])) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// minFunc devuelve el menor de dos enteros.
// Uso en template: {{ min 5 10 }} -> 5
func minFunc(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxFunc devuelve el mayor de dos enteros.
// Uso en template: {{ max 5 10 }} -> 10
func maxFunc(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// formatDate formatea un objeto time.Time a un string con el layout dado.
// Define un formato por defecto fácil de recordar.
const DefaultDateFormat = "02/01/2006"

// formatDate formatea una fecha. Si no se especifica un layout,
// usa el formato "DD/MM/AAAA".
// Uso en template: {{ .MiFecha | formatDate "02 Jan 2006" }}
func formatDate(t time.Time, layout ...string) string {
	// Si se proveyó un layout y no está vacío, úsalo.
	if len(layout) > 0 && layout[0] != "" {
		return t.Format(layout[0])
	}
	// De lo contrario, usa el formato por defecto.
	return t.Format(DefaultDateFormat)
}

func dict(v ...interface{}) map[string]interface{} {
	if len(v)%2 != 0 {
		panic("dict requiere número par de argumentos")
	}
	m := make(map[string]interface{}, len(v)/2)
	for i := 0; i < len(v); i += 2 {
		key, ok := v[i].(string)
		if !ok {
			panic("las claves deben ser string")
		}
		m[key] = v[i+1]
	}
	return m
}

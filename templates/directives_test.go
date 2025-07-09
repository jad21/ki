package templates

import (
	"html/template"
	"testing"
	"time"
)

// TestMergeFuncMaps verifica la correcta fusión de mapas de funciones.
func TestMergeFuncMaps(t *testing.T) {
	t.Parallel()
	map1 := template.FuncMap{
		"funcA": func() string { return "A" },
		"funcB": func() string { return "B" },
	}
	map2 := template.FuncMap{
		"funcB": func() string { return "B_new" }, // Sobrescribe funcB
		"funcC": func() string { return "C" },
	}

	merged := mergeFuncMaps(map1, map2)

	// Verificar tamaño
	if len(merged) != 3 {
		t.Errorf("Se esperaba un mapa fusionado de tamaño 3, se obtuvo %d", len(merged))
	}

	// Verificar funcA
	if val, ok := merged["funcA"]; !ok || val.(func() string)() != "A" {
		t.Errorf("funcA no se fusionó correctamente")
	}

	// Verificar funcB (sobrescrita)
	if val, ok := merged["funcB"]; !ok || val.(func() string)() != "B_new" {
		t.Errorf("funcB no se sobrescribió correctamente")
	}

	// Verificar funcC
	if val, ok := merged["funcC"]; !ok || val.(func() string)() != "C" {
		t.Errorf("funcC no se fusionó correctamente")
	}
}

// TestToJSON verifica la conversión a JSON.
func TestToJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    any
		expected string
		hasError bool
	}{
		{
			name:     "objeto simple",
			input:    struct{ Name string }{Name: "Test"},
			expected: `{"Name":"Test"}`,
			hasError: false,
		},
		{
			name:     "slice de strings",
			input:    []string{"a", "b"},
			expected: `["a","b"]`,
			hasError: false,
		},
		{
			name:     "nil",
			input:    nil,
			expected: `null`,
			hasError: false,
		},
		{
			name:     "tipo que causa error", // un canal no puede ser marshaleado a JSON
			input:    make(chan int),
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := toJSON(tt.input)
			if tt.hasError {
				if err == nil {
					t.Errorf("toJSON() para %s se esperaba un error, pero no se obtuvo ninguno", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("toJSON() para %s retornó un error inesperado: %v", tt.name, err)
				}
				if string(got) != tt.expected {
					t.Errorf("toJSON() para %s got = %q, want %q", tt.name, got, tt.expected)
				}
			}
		})
	}
}

// TestToSnakeCase verifica la conversión de CamelCase a snake_case.
func TestToSnakeCase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"CamelCase simple", "MiVariable", "mi_variable"},
		{"CamelCase con acrónimo", "HTTPRequest", "http_request"},
		{"PascalCase", "OtraVariableDePrueba", "otra_variable_de_prueba"},
		{"SnakeCase ya existente", "ya_es_snake_case", "ya_es_snake_case"},
		{"string vacío", "", ""},
		{"un carácter", "A", "a"},
		{"todo mayúsculas", "API", "api"},
		{"números incluidos", "MiVariable123ID", "mi_variable123_id"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toSnakeCase(tt.input)
			if got != tt.expected {
				t.Errorf("toSnakeCase() para %q got = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestToCamelCase verifica la conversión de snake_case a UpperCamelCase.
func TestToCamelCase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"snake_case simple", "mi_variable", "MiVariable"},
		{"snake_case con números", "variable_123_id", "Variable123Id"},
		{"string vacío", "", ""},
		{"sin underscores", "singles", "Singles"},
		{"múltiples underscores", "multi__underscore_test", "MultiUnderscoreTest"}, // Esto puede ser un caso de borde o comportamiento esperado
		{"solo underscore", "_", ""},                                               // Split por "_" daría ["", ""], el bucle lo manejaría a ""
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := toCamelCase(tt.input)
			if got != tt.expected {
				t.Errorf("toCamelCase() para %q got = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestMinFunc verifica la función minFunc.
func TestMinFunc(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"a menor que b", 5, 10, 5},
		{"b menor que a", 10, 5, 5},
		{"iguales", 7, 7, 7},
		{"números negativos", -5, -10, -10},
		{"cero", 0, 5, 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := minFunc(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("minFunc(%d, %d) got = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

// TestMaxFunc verifica la función maxFunc.
func TestMaxFunc(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"a mayor que b", 10, 5, 10},
		{"b mayor que a", 5, 10, 10},
		{"iguales", 7, 7, 7},
		{"números negativos", -5, -10, -5},
		{"cero", 0, -5, 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := maxFunc(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("maxFunc(%d, %d) got = %d, want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}

// TestFormatDate verifica la función formatDate.
func TestFormatDate(t *testing.T) {
	t.Parallel()
	testTime := time.Date(2023, time.November, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name      string
		inputTime time.Time
		layout    []string
		expected  string
	}{
		{
			name:      "formato por defecto",
			inputTime: testTime,
			layout:    nil,
			expected:  "15/11/2023",
		},
		{
			name:      "formato personalizado (DD MMM YYYY)",
			inputTime: testTime,
			layout:    []string{"02 Jan 2006"},
			expected:  "15 Nov 2023",
		},
		{
			name:      "formato personalizado (YYYY-MM-DD)",
			inputTime: testTime,
			layout:    []string{"2006-01-02"},
			expected:  "2023-11-15",
		},
		{
			name:      "layout vacío en el slice",
			inputTime: testTime,
			layout:    []string{""},
			expected:  "15/11/2023", // debería usar el por defecto
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got string
			if tt.layout == nil {
				got = formatDate(tt.inputTime)
			} else {
				got = formatDate(tt.inputTime, tt.layout...)
			}

			if got != tt.expected {
				t.Errorf("formatDate() para %s got = %q, want %q", tt.name, got, tt.expected)
			}
		})
	}
}

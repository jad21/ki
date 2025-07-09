package templates

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

// Helper para crear archivos temporales para las pruebas.
// Helper para crear archivos temporales para las pruebas.
func createTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	filePath := filepath.Join(dir, name)

	// **MODIFICACIÓN AQUÍ:** Asegurarse de que el directorio del archivo exista
	fileDir := filepath.Dir(filePath)
	if err := os.MkdirAll(fileDir, 0755); err != nil {
		t.Fatalf("No se pudo crear el directorio %s: %v", fileDir, err)
	}

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("No se pudo crear el archivo temporal %s: %v", filePath, err)
	}
	return filePath
}

// TestNew verifica la creación de un nuevo registro de plantillas.
func TestNew(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	// Archivos de muestra
	createTempFile(t, tmp, "header.html", "<h1>{{.Title}}</h1>")
	createTempFile(t, tmp, "pages/index.html", "<p>Hola mundo</p>")
	createTempFile(t, tmp, "pages/about.gotpl", `{{define "about_page"}}About Us{{end}}`)
	createTempFile(t, tmp, "partials/_footer.html", `<footer>© {{formatDate .Year "2006"}}</footer>`)

	cases := []struct {
		name string
		opts []Option
		want []string
	}{
		{
			"Dir()",
			[]Option{Dir(tmp)},
			[]string{
				"root",
				"header.html",
				"pages.index.html",
				"pages.about.gotpl", // contenedor
				"about_page",        // define interno
				"partials._footer.html",
			},
		},
		{
			"DirFS()",
			[]Option{DirFS(os.DirFS(tmp))},
			[]string{
				"root",
				"header.html",
				"pages.index.html",
				"pages.about.gotpl",
				"about_page",
				"partials._footer.html",
			},
		},
		{
			"solo .html",
			[]Option{DirFS(os.DirFS(tmp)), Suffix(".html")},
			[]string{
				"root",
				"header.html",
				"pages.index.html",
				"partials._footer.html",
			},
		},
		{
			"FuncMap extra",
			[]Option{
				DirFS(os.DirFS(tmp)),
				FuncMap(template.FuncMap{"customFunc": func() string { return "hi" }}),
			},
			[]string{
				"root",
				"header.html",
				"pages.index.html",
				"pages.about.gotpl",
				"about_page",
				"partials._footer.html",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			reg, err := New(tc.opts...)
			if err != nil {
				t.Fatalf("New(): %v", err)
			}

			got := reg.ListTemplates()
			sort.Strings(got)
			sort.Strings(tc.want)

			if len(got) != len(tc.want) {
				t.Fatalf("\ncount: got %d, want %d\nGot:  %v\nWant: %v",
					len(got), len(tc.want), got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("\nidx %d: got %q, want %q\nGot:  %v\nWant: %v",
						i, got[i], tc.want[i], got, tc.want)
				}
			}
		})
	}
}

// TestCommonDir verifica la función commonDir.
func TestCommonDir(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		paths    []string
		expected string
	}{
		{
			name:     "ruta común simple",
			paths:    []string{"/a/b/c/file1.txt", "/a/b/d/file2.txt", "/a/b/e/file3.txt"},
			expected: "/a/b",
		},
		{
			name:     "sin ruta común (raíz)",
			paths:    []string{"/a/file1.txt", "/b/file2.txt"},
			expected: string(os.PathSeparator),
		},
		{
			name:     "un solo archivo",
			paths:    []string{"/a/b/c/file1.txt"},
			expected: "/a/b/c",
		},
		{
			name:     "archivos en el mismo directorio",
			paths:    []string{"/a/b/file1.txt", "/a/b/file2.txt"},
			expected: "/a/b",
		},
		{
			name:     "lista vacía",
			paths:    []string{},
			expected: "",
		},
		{
			name:     "rutas relativas",
			paths:    []string{"dir1/sub/file1.txt", "dir1/sub/sub2/file2.txt"},
			expected: "dir1/sub",
		},
		{
			name:     "archivos en el directorio raíz",
			paths:    []string{"file1.txt", "file2.txt"},
			expected: ".",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Normalizar las rutas esperadas para el sistema operativo actual
			if tt.expected != "" && tt.expected != "." && tt.expected != string(os.PathSeparator) {
				tt.expected = filepath.FromSlash(tt.expected)
			}
			for i := range tt.paths {
				tt.paths[i] = filepath.FromSlash(tt.paths[i])
			}

			got := commonDir(tt.paths)
			if got != tt.expected {
				t.Errorf("commonDir(%v) got = %q, want %q", tt.paths, got, tt.expected)
			}
		})
	}
}

// TestListTemplates verifica que ListTemplates devuelve los nombres correctos de las plantillas.
func TestListTemplates(t *testing.T) {
	t.Parallel()
	tempDir, err := os.MkdirTemp("", "list_templates_test_")
	if err != nil {
		t.Fatalf("No se pudo crear el directorio temporal: %v", err)
	}
	defer os.RemoveAll(tempDir)

	createTempFile(t, tempDir, "tmpl1.html", "Hello from tmpl1")
	createTempFile(t, tempDir, "sub/tmpl2.html", "Hello from tmpl2")
	createTempFile(t, tempDir, "sub/tmpl3.gotpl", "{{define \"my_defined_tmpl\"}}Defined template{{end}}")

	reg, err := New(DirFS(os.DirFS(tempDir)))
	if err != nil {
		t.Fatalf("New() retornó un error inesperado: %v", err)
	}

	expected := []string{"my_defined_tmpl", "root", "sub.tmpl2.html", "tmpl1.html", "sub.tmpl3.gotpl"} // "root" es el template que envuelve todo

	// Obtener los nombres y ordenarlos para una comparación consistente
	got := reg.ListTemplates()
	// La función ListTemplates interna retorna los nombres de forma desordenada
	// por lo que debemos ordenarlos para la comparación
	sortStrings(got)
	sortStrings(expected)

	if len(got) != len(expected) {
		t.Errorf("ListTemplates() retornó %d nombres, se esperaban %d. \nGot: %v, \nExpected: %v", len(got), len(expected), got, expected)
		return
	}

	for i, name := range got {
		if name != expected[i] {
			t.Errorf("En la posición %d, got = %q, want %q", i, name, expected[i])
		}
	}
}

// sortStrings es una función auxiliar para ordenar slices de strings.
func sortStrings(s []string) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// TestExecuteTemplate verifica la ejecución de una plantilla.
func TestExecuteTemplate(t *testing.T) {
	t.Parallel()
	tempDir, err := os.MkdirTemp("", "execute_template_test_")
	if err != nil {
		t.Fatalf("No se pudo crear el directorio temporal: %v", err)
	}
	defer os.RemoveAll(tempDir)

	createTempFile(t, tempDir, "greeting.html", "Hello, {{.Name}}!")
	createTempFile(t, tempDir, "complex/data.html", "Current year: {{formatDate .Year \"2006\"}}")
	createTempFile(t, tempDir, "error.html", "{{.NonExistentField}}") // Plantilla que causará error de ejecución si el campo no existe

	reg, err := New(DirFS(os.DirFS(tempDir)))
	if err != nil {
		t.Fatalf("New() retornó un error inesperado: %v", err)
	}

	tests := []struct {
		name        string
		tmplName    string
		data        any
		expected    string
		expectedErr bool
	}{
		{
			name:        "ejecución básica",
			tmplName:    "greeting.html",
			data:        struct{ Name string }{Name: "World"},
			expected:    "Hello, World!",
			expectedErr: false,
		},
		{
			name:        "plantilla anidada con función",
			tmplName:    "complex.data.html",
			data:        struct{ Year time.Time }{Year: time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)},
			expected:    "Current year: 2025",
			expectedErr: false,
		},
		{
			name:        "plantilla inexistente",
			tmplName:    "non_existent.html",
			data:        nil,
			expected:    "",
			expectedErr: true,
		},
		{
			name:        "error en ejecución de plantilla (campo inexistente)",
			tmplName:    "error.html",
			data:        struct{ Name string }{Name: "Test"},
			expected:    "",
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := reg.ExecuteTemplate(&buf, tt.tmplName, tt.data)

			if tt.expectedErr {
				if err == nil {
					t.Errorf("ExecuteTemplate() para %s se esperaba un error, pero no se obtuvo ninguno", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("ExecuteTemplate() para %s retornó un error inesperado: %v", tt.name, err)
				}
				if buf.String() != tt.expected {
					t.Errorf("ExecuteTemplate() para %s got = %q, want %q", tt.name, buf.String(), tt.expected)
				}
			}
		})
	}
}

// TestFuncs verifica que la función Funcs del registro actualice el FuncMap del Base template.
func TestFuncs(t *testing.T) {
	t.Parallel()
	reg, err := New() // Crear un registro con opciones por defecto para simplificar
	if err != nil {
		// Esto puede ocurrir si el directorio 'templates' por defecto no existe.
		// Para esta prueba específica, necesitamos un registro que funcione.
		// Se puede crear un directorio temporal para asegurar la operación si se desea.
		tempDir, err := os.MkdirTemp("", "test_funcs_")
		if err != nil {
			t.Fatalf("No se pudo crear el directorio temporal: %v", err)
		}
		defer os.RemoveAll(tempDir)
		reg, err = New(DirFS(os.DirFS(tempDir)))
		if err != nil {
			t.Fatalf("New() retornó un error inesperado incluso con temp dir: %v", err)
		}
	}

	customFuncMap := template.FuncMap{
		"newFunc": func() string { return "nueva función" },
		"json":    func(any) (template.JS, error) { return "sobreescrito", nil }, // Sobrescribir una función existente
	}

	// Aplicar el nuevo FuncMap
	updatedTmpl := reg.Funcs(customFuncMap)

	if updatedTmpl != reg.Base {
		t.Errorf("Funcs() no retornó el mismo puntero al template base")
	}

	// Intentar obtener las funciones del template base y verificar
	// Note: No hay un método directo para obtener el FuncMap de un *template.Template
	// después de haberlo establecido. La forma de verificar es intentar usar las funciones.

	// Crear una plantilla simple para probar las funciones
	tmplStr := `{{newFunc}} {{json .Data}}`
	// tmpl, err := template.New("testFuncs").Funcs(reg.Base.Funcs()).Parse(tmplStr)
	tmpl, err := reg.Parse(tmplStr)
	if err != nil {
		t.Fatalf("No se pudo parsear la plantilla de prueba: %v", err)
	}

	data := struct{ Data string }{Data: "some data"}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		t.Fatalf("No se pudo ejecutar la plantilla de prueba: %v", err)
	}

	expectedOutput := "nueva función sobreescrito" // El "json" original hubiera intentado parsear .Data, pero ahora está sobrescrito.
	if buf.String() != expectedOutput {
		t.Errorf("La ejecución de la plantilla con Funcs() got = %q, want %q", buf.String(), expectedOutput)
	}
}

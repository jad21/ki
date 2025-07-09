package templates

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jad21/ki/env"
)

// Registry contiene todos los templates parseados y listos para usar.
// Se crea una vez al iniciar la aplicación.
type Registry struct {
	Base               *template.Template
	Dir                FileReader
	FuncMap            template.FuncMap
	Suffixes           []string
	Tree               map[string]string
	hasDefineDirective map[string]bool
}
type FileReader interface {
	Open(string) (fs.File, error)
}
type options struct {
	Dir      FileReader
	FuncMap  template.FuncMap
	Suffixes []string
}
type Option func(o *options)

var (
	definePattern = regexp.MustCompile(`{{\s*(define|block)\b.+}}`)

	defaultOptions = options{
		Dir: os.DirFS(env.GetEnvVar("TEMPLATES", "templates")),
	}
)

func Dir(dir string) Option {
	return func(o *options) {
		o.Dir = os.DirFS(dir)

	}
}
func DirFS(reader FileReader) Option {
	return func(o *options) {
		o.Dir = reader
	}
}
func FuncMap(fn template.FuncMap) Option {
	return func(o *options) {
		o.FuncMap = fn
	}
}
func Suffix(s ...string) Option {
	return func(o *options) {
		o.Suffixes = append(o.Suffixes, s...)
	}
}

func New(opt ...Option) (*Registry, error) {
	rootTmpl := template.New("root")
	opts := defaultOptions
	for _, o := range opt {
		o(&opts)
	}

	r := &Registry{
		FuncMap:            mergeFuncMaps(opts.FuncMap, FuncMapDefault()),
		Base:               rootTmpl,
		Dir:                opts.Dir,
		Suffixes:           opts.Suffixes,
		Tree:               make(map[string]string, 0),
		hasDefineDirective: make(map[string]bool, 0),
	}

	paths := DiscoverFS(r.Dir, r.Suffixes...)
	rootDir := commonDir(paths)
	for _, absPath := range paths {
		file, err := r.Dir.Open(absPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", absPath, err)
		}
		defer file.Close() // Asegura que el archivo se cierre
		b, err := io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", absPath, err)
		}
		rel, _ := filepath.Rel(rootDir, absPath)
		logicalName := strings.ReplaceAll(rel, string(os.PathSeparator), ".")
		content := string(b)
		r.Tree[logicalName] = content
		r.hasDefineDirective[logicalName] = hasDefineDirective(content)
		_, err = rootTmpl.New(logicalName).Funcs(r.FuncMap).Parse(content)
		if err != nil {
			// os.WriteFile(logicalName, []byte(content), 0644)
			return nil, handleTemplateError(err, content)
		}
	}

	r.Base = rootTmpl
	return r, nil

}

// ListTemplates devuelve una lista de los nombres lógicos de todos los templates cargados.
func (r *Registry) ListTemplates() []string {
	clone, err := r.Base.Clone()
	if err != nil {
		return nil
	}
	var names []string
	for _, t := range clone.Templates() {
		names = append(names, t.Name())
	}
	return names
}

func (r *Registry) ExecuteTemplate(w io.Writer, name string, data any) error {
	// siempre usamos el clone
	clone, err := r.Base.Clone()
	if err != nil {
		return err
	}
	tmpl := clone

	if v, ok := r.Tree[name]; ok && r.hasDefineDirective[name] {
		// siempre pareamos para sobrescribir el define, si los hay
		if tmpl, err = clone.New(name).Parse(v); err != nil {
			return err
		}
	}
	return tmpl.ExecuteTemplate(w, name, data)
}

func (r *Registry) Execute(w io.Writer, data any) error {
	return r.Base.Execute(w, data)
}
func (r *Registry) Parse(text string) (*template.Template, error) {
	return r.Base.Parse(text)
}

func (r *Registry) Funcs(funcMap template.FuncMap) *template.Template {
	return r.Base.Funcs(funcMap)
	// return r
}

// -------------------- helpers -------------------------

// Detecta si es un componente con {{define Name}}

func hasDefineDirective(source string) bool {
	return definePattern.FindString(source) != ""

}

// handleTemplateError procesa un error de plantilla, intentando extraer el contexto
// de la fuente de la plantilla para ofrecer un mensaje de error más útil.
// Recibe el error original de la plantilla y el código fuente de la plantilla.
// Devuelve un nuevo error que incluye el contexto o el error original si no se puede procesar.
func handleTemplateError(err error, source string) error {
	// El formato del error suele ser "template: [nombre]:[línea]:[columna]: [mensaje]"
	// Dividimos el string por ":" para intentar extraer el número de línea.
	parts := strings.SplitN(err.Error(), ":", 4)

	// Si tenemos suficientes partes, el segundo elemento debería ser la línea.
	if len(parts) >= 3 {
		line, convErr := strconv.Atoi(parts[2]) // Asumiendo que el formato es `[nombre]:[linea]:[columna]`
		if convErr == nil {
			// Éxito al obtener la línea. Ahora construimos el error con contexto.
			var errSb strings.Builder
			errSb.WriteString("error al procesar la plantilla:\n")
			errSb.WriteString(err.Error()) // Incluir el error original
			errSb.WriteString("\n\n--- file ---\n")

			lines := strings.Split(source, "\n")
			lineNumber := line - 1 // El error es 1-based, el slice es 0-based.

			// Definimos una "ventana" de 2 líneas antes y 2 después del error.
			start := max(lineNumber-4, 0)
			end := lineNumber + 4 // +2 para las líneas posteriores y +1 para el rango de slice.
			if end > len(lines) {
				end = len(lines)
			}

			// Imprimimos las líneas de contexto, destacando la del error.
			for i := start; i < end; i++ {
				if i == lineNumber {
					errSb.WriteString(fmt.Sprintf(">> %4d | %s\n", i+1, lines[i]))
				} else {
					errSb.WriteString(fmt.Sprintf("   %4d | %s\n", i+1, lines[i]))
				}
			}

			// Devolvemos el nuevo error enriquecido.
			return fmt.Errorf("%s", errSb.String())
		}
	}

	// Si no pudimos decodificar el error, devolvemos el original.
	return err
}

// commonDir retorna el directorio común para varios paths absolutos.
func commonDir(paths []string) string {
	if len(paths) == 0 {
		return ""
	}

	// Limpiamos y obtenemos el directorio base del primer path.
	// Este será nuestro candidato inicial para el directorio común.
	commonPath := filepath.Clean(paths[0])

	// Si el primer path es un archivo en el directorio actual, su "directorio" es "."
	if !filepath.IsAbs(commonPath) && !strings.Contains(commonPath, string(filepath.Separator)) {
		commonPath = "."
	} else {
		commonPath = filepath.Dir(commonPath)
	}

	// Iteramos sobre el resto de los paths.
	for _, p := range paths[1:] {
		cleanedP := filepath.Clean(p)

		// Mientras el path actual no tenga como prefijo nuestro commonPath,
		// acortamos commonPath hacia su padre.
		// También nos aseguramos de no acortar commonPath si ya es la raíz
		// o si es ".", para evitar bucles infinitos o resultados incorrectos.
		for !strings.HasPrefix(cleanedP, commonPath) && commonPath != string(filepath.Separator) && commonPath != "." {
			commonPath = filepath.Dir(commonPath)
		}

		// Si commonPath se reduce a la raíz y aún no es prefijo,
		// pero los paths son absolutos, establecemos commonPath a la raíz.
		// Si no son absolutos y ya no hay prefijo común, es "."
		if !strings.HasPrefix(cleanedP, commonPath) {
			if filepath.IsAbs(cleanedP) {
				commonPath = string(filepath.Separator)
			} else {
				commonPath = "."
			}
		}
	}

	// Si al final commonPath es solo el separador de ruta y el path original
	// no era absoluto, o si es un string vacío, el directorio común es "."
	if commonPath == "" || (commonPath == string(filepath.Separator) && !filepath.IsAbs(paths[0])) {
		return "."
	}

	return commonPath
}

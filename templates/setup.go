package templates

import (
	"ego/models"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// PageData holds data passed to templates
type PageData struct {
	Title       string
	User        *models.User
	Data        interface{}
	CompanyName string
	Page        string
}

// TemplateStore holds all parsed templates
type TemplateStore struct {
	templates map[string]*template.Template
	funcMap   template.FuncMap
}

var store *TemplateStore

// Init initializes the template store
func Init() {
	store = &TemplateStore{
		templates: make(map[string]*template.Template),
		funcMap: template.FuncMap{
			"seq": func(n int) []int {
				s := make([]int, n)
				for i := 0; i < n; i++ {
					s[i] = i
				}
				return s
			},
			"add": func(a, b int) int {
				return a + b
			},
			"sub": func(a, b int) int {
				return a - b
			},
			"dict": func(values ...interface{}) map[string]interface{} {
				dict := make(map[string]interface{})
				for i := 0; i < len(values); i += 2 {
					if i+1 < len(values) {
						key, ok := values[i].(string)
						if ok {
							dict[key] = values[i+1]
						}
					}
				}
				return dict
			},
			"safeHTML": func(s string) template.HTML {
				return template.HTML(s)
			},
			"lower": strings.ToLower,
			"upper": strings.ToUpper,
		},
	}

	// Parse all template files
	err := filepath.Walk("templates", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		tmpl := template.New(info.Name()).Funcs(store.funcMap)
		tmpl, err = tmpl.Parse(string(content))
		if err != nil {
			return err
		}

		store.templates[path] = tmpl
		return nil
	})

	if err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}

	log.Printf("Loaded %d templates", len(store.templates))
}

// Render renders a template
func Render(w http.ResponseWriter, layout, page string, data interface{}) {
	// Try to find the template by layout+page combination
	templateKey := findTemplateKey(layout, page)
	if templateKey == "" {
		log.Printf("Template not found for layout=%s page=%s", layout, page)
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	tmpl, ok := store.templates[templateKey]
	if !ok {
		log.Printf("Template key %s not found in store", templateKey)
		return
	}

	err := tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
	}
}

// findTemplateKey finds the template key for a given layout and page
func findTemplateKey(layout, page string) string {
	// Try exact match first
	key := filepath.Join("templates", "_pages", layout+"-"+page+".html")
	if _, ok := store.templates[key]; ok {
		return key
	}

	// Try layout-specific pages
	key = filepath.Join("templates", "_pages", layout, page+".html")
	if _, ok := store.templates[key]; ok {
		return key
	}

	// Try sections
	key = filepath.Join("templates", "sections", page+".html")
	if _, ok := store.templates[key]; ok {
		return key
	}

	// Try direct page name
	key = filepath.Join("templates", "_pages", page+".html")
	if _, ok := store.templates[key]; ok {
		return key
	}

	return ""
}

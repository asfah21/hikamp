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
	CurrentPath string
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
			"div": func(a, b int) int {
				if b == 0 {
					return 0
				}
				return a / b
			},
			"mod": func(a, b int) int {
				if b == 0 {
					return 0
				}
				return a % b
			},
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

// RenderPartial renders only the page template content without layout (for HTMX modals)
func RenderPartial(w http.ResponseWriter, layout, page string, data interface{}) {
	// Find page template
	pageKey := findTemplateKey(layout, page)
	if pageKey == "" {
		log.Printf("Page template not found for layout=%s page=%s", layout, page)
		http.Error(w, "Page template not found", http.StatusInternalServerError)
		return
	}

	tmpl, ok := store.templates[pageKey]
	if !ok {
		log.Printf("Page template key %s not found in store", pageKey)
		return
	}

	err := tmpl.ExecuteTemplate(w, "content", data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Template execution error: "+err.Error(), http.StatusInternalServerError)
	}
}

// Render renders a template by combining layout + page
func Render(w http.ResponseWriter, layout, page string, data interface{}) {
	// Find layout template
	layoutKey := filepath.Join("templates", "layout-"+layout+".html")
	if _, ok := store.templates[layoutKey]; !ok {
		log.Printf("Layout template not found: %s", layoutKey)
		http.Error(w, "Layout template not found", http.StatusInternalServerError)
		return
	}

	// Find page template
	pageKey := findTemplateKey(layout, page)
	if pageKey == "" {
		log.Printf("Page template not found for layout=%s page=%s", layout, page)
		http.Error(w, "Page template not found", http.StatusInternalServerError)
		return
	}

	if _, ok := store.templates[pageKey]; !ok {
		log.Printf("Page template key %s not found in store", pageKey)
		return
	}

	// Merge layout + page into one template
	merged := template.New("merged").Funcs(store.funcMap)

	// Parse layout content directly
	layoutContent, err := os.ReadFile(layoutKey)
	if err != nil {
		log.Printf("Failed to read layout file: %v", err)
		http.Error(w, "Failed to read layout", http.StatusInternalServerError)
		return
	}

	// Parse page content directly
	pageContent, err := os.ReadFile(pageKey)
	if err != nil {
		log.Printf("Failed to read page file: %v", err)
		http.Error(w, "Failed to read page", http.StatusInternalServerError)
		return
	}

	// Combine layout + page content
	combined := string(layoutContent) + "\n" + string(pageContent)

	merged, err = merged.Parse(combined)
	if err != nil {
		log.Printf("Template parse error: %v", err)
		http.Error(w, "Template parse error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	err = merged.Execute(w, data)
	if err != nil {
		log.Printf("Template execution error: %v", err)
		http.Error(w, "Template execution error: "+err.Error(), http.StatusInternalServerError)
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

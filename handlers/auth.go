package handlers

import (
	"context"
	"ego/models"
	"ego/templates"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// HandlePublic redirects to login page
func HandlePublic(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// HandleLogin handles login page and form submission
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")

		// Authenticate user
		user, err := authenticateUser(username, password)
		if err != nil {
			if isHTMXRequest(r) {
				templates.RenderPartial(w, "", "login-form", map[string]interface{}{
					"Error": "Invalid username or password",
				})
			} else {
				templates.Render(w, "auth", "login", map[string]interface{}{
					"Error": "Invalid username or password",
				})
			}
			return
		}

		// Create session
		sessionToken, err := createSession(user.ID)
		if err != nil {
			log.Printf("Failed to create session: %v", err)
			if isHTMXRequest(r) {
				templates.RenderPartial(w, "", "login-form", map[string]interface{}{
					"Error": "Failed to create session",
				})
			} else {
				templates.Render(w, "auth", "login", map[string]interface{}{
					"Error": "Failed to create session",
				})
			}
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    sessionToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   false,     // Set to true in production
			MaxAge:   86400 * 7, // 7 days
		})

		w.Header().Set("HX-Redirect", "/admin/dashboard")
		w.WriteHeader(http.StatusOK)
		return
	}

	templates.Render(w, "auth", "login", nil)
}

// HandleLogout handles logout
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		deleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// RequireAuth is middleware that checks for a valid session
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			if isHTMXRequest(r) {
				w.Header().Set("HX-Redirect", "/login")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		userID, err := getSessionUserID(cookie.Value)
		if err != nil {
			if isHTMXRequest(r) {
				w.Header().Set("HX-Redirect", "/login")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Get user and add to context
		user, err := getUserByID(userID)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Add user to request context
		ctx := r.Context()
		ctx = contextWithUser(ctx, user)
		next(w, r.WithContext(ctx))
	}
}

// isHTMXRequest checks if the request is an HTMX request
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// Simple in-memory session store
var sessions = make(map[string]int)

func createSession(userID int) (string, error) {
	token := generateSessionToken()
	sessions[token] = userID
	return token, nil
}

func getSessionUserID(token string) (int, error) {
	userID, ok := sessions[token]
	if !ok {
		return 0, http.ErrNoCookie
	}
	return userID, nil
}

func deleteSession(token string) {
	delete(sessions, token)
}

func generateSessionToken() string {
	return strings.ReplaceAll(uuid(), "-", "")
}

func uuid() string {
	// Simple UUID-like generation
	return "session-" + fmt.Sprintf("%d", time.Now().UnixNano())
}

// User storage (simple in-memory for now)
var users = []models.User{
	{ID: 1, Username: "admin", Password: "admin123", Name: "Administrator", Role: "admin", Enabled: true},
}

func authenticateUser(username, password string) (*models.User, error) {
	for _, u := range users {
		if u.Username == username && u.Password == password && u.Enabled {
			return &u, nil
		}
	}
	return nil, http.ErrNoCookie
}

func getUserByID(id int) (*models.User, error) {
	for _, u := range users {
		if u.ID == id {
			return &u, nil
		}
	}
	return nil, http.ErrNoCookie
}

func contextWithUser(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, "user", user)
}

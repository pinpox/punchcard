package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// User represents a user in the system
type User struct {
	ID          int       `json:"id"`
	OIDCSubject string    `json:"oidc_subject"` // The "sub" claim from OIDC token
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	LastLoginAt time.Time `json:"last_login_at"`
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	UserID    int       `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// OIDCConfig holds OIDC configuration
type OIDCConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// AuthService handles authentication operations
type AuthService struct {
	db       *sql.DB
	config   OIDCConfig
	provider *oidc.Provider
	oauth2   oauth2.Config
}

// NewAuthService creates a new authentication service
func NewAuthService(db *sql.DB, config OIDCConfig) (*AuthService, error) {
	ctx := context.Background()
	
	provider, err := oidc.NewProvider(ctx, config.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	oauth2Config := oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return &AuthService{
		db:       db,
		config:   config,
		provider: provider,
		oauth2:   oauth2Config,
	}, nil
}

// GenerateState generates a random state string for OIDC
func (a *AuthService) GenerateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// GetAuthURL returns the OIDC authorization URL
func (a *AuthService) GetAuthURL(state string) string {
	return a.oauth2.AuthCodeURL(state)
}

// ExchangeCode exchanges authorization code for tokens and creates/updates user
func (a *AuthService) ExchangeCode(ctx context.Context, code string) (*User, error) {
	token, err := a.oauth2.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in token response")
	}

	verifier := a.provider.Verifier(&oidc.Config{ClientID: a.config.ClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	var claims struct {
		Subject           string `json:"sub"`
		Email            string `json:"email"`
		Name             string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
	}
	
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("failed to parse claims: %w", err)
	}

	// Use preferred_username if available, otherwise fall back to name
	username := claims.PreferredUsername
	if username == "" {
		username = claims.Name
	}

	// Create or update user
	user, err := a.createOrUpdateUser(claims.Subject, claims.Email, username)
	if err != nil {
		return nil, fmt.Errorf("failed to create/update user: %w", err)
	}

	return user, nil
}

// createOrUpdateUser creates a new user or updates existing one
func (a *AuthService) createOrUpdateUser(subject, email, username string) (*User, error) {
	// Check if user exists
	var user User
	err := a.db.QueryRow(`
		SELECT id, oidc_subject, email, name, created_at, last_login_at 
		FROM users WHERE oidc_subject = ?
	`, subject).Scan(&user.ID, &user.OIDCSubject, &user.Email, &user.Name, &user.CreatedAt, &user.LastLoginAt)

	if err == sql.ErrNoRows {
		// Create new user
		now := time.Now()
		result, err := a.db.Exec(`
			INSERT INTO users (oidc_subject, email, name, created_at, last_login_at)
			VALUES (?, ?, ?, ?, ?)
		`, subject, email, username, now, now)
		if err != nil {
			return nil, err
		}

		id, err := result.LastInsertId()
		if err != nil {
			return nil, err
		}

		user = User{
			ID:          int(id),
			OIDCSubject: subject,
			Email:       email,
			Name:        username,
			CreatedAt:   now,
			LastLoginAt: now,
		}
	} else if err != nil {
		return nil, err
	} else {
		// Update existing user's last login
		now := time.Now()
		_, err = a.db.Exec(`
			UPDATE users SET email = ?, name = ?, last_login_at = ?
			WHERE id = ?
		`, email, username, now, user.ID)
		if err != nil {
			return nil, err
		}
		user.Email = email
		user.Name = username
		user.LastLoginAt = now
	}

	return &user, nil
}

// CreateSession creates a new session for the user
func (a *AuthService) CreateSession(userID int) (*Session, error) {
	sessionID, err := a.GenerateState() // Reuse the state generation function
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(24 * time.Hour * 7) // 7 days
	now := time.Now()

	_, err = a.db.Exec(`
		INSERT INTO sessions (id, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, sessionID, userID, expiresAt, now)
	if err != nil {
		return nil, err
	}

	return &Session{
		ID:        sessionID,
		UserID:    userID,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

// GetUserFromSession retrieves user by session ID
func (a *AuthService) GetUserFromSession(sessionID string) (*User, error) {
	var user User
	var session Session
	
	err := a.db.QueryRow(`
		SELECT s.id, s.user_id, s.expires_at, s.created_at,
		       u.id, u.oidc_subject, u.email, u.name, u.created_at, u.last_login_at
		FROM sessions s
		JOIN users u ON s.user_id = u.id
		WHERE s.id = ? AND s.expires_at > ?
	`, sessionID, time.Now()).Scan(
		&session.ID, &session.UserID, &session.ExpiresAt, &session.CreatedAt,
		&user.ID, &user.OIDCSubject, &user.Email, &user.Name, &user.CreatedAt, &user.LastLoginAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// DeleteSession deletes a session (logout)
func (a *AuthService) DeleteSession(sessionID string) error {
	_, err := a.db.Exec("DELETE FROM sessions WHERE id = ?", sessionID)
	return err
}

// CleanupExpiredSessions removes expired sessions
func (a *AuthService) CleanupExpiredSessions() error {
	_, err := a.db.Exec("DELETE FROM sessions WHERE expires_at < ?", time.Now())
	return err
}

// AuthMiddleware provides authentication middleware
func (a *AuthService) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for login/callback routes
		if strings.HasPrefix(r.URL.Path, "/login") || 
		   strings.HasPrefix(r.URL.Path, "/callback") ||
		   strings.HasPrefix(r.URL.Path, "/static/") {
			next.ServeHTTP(w, r)
			return
		}

		sessionCookie, err := r.Cookie("session")
		if err != nil {
			a.redirectToLogin(w, r)
			return
		}

		user, err := a.GetUserFromSession(sessionCookie.Value)
		if err != nil {
			a.redirectToLogin(w, r)
			return
		}

		// Add user to request context
		ctx := context.WithValue(r.Context(), "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// redirectToLogin redirects user to login page
func (a *AuthService) redirectToLogin(w http.ResponseWriter, r *http.Request) {
	// Store original URL in session for redirect after login
	returnURL := r.URL.String()
	if returnURL != "" && returnURL != "/" {
		http.SetCookie(w, &http.Cookie{
			Name:     "return_url",
			Value:    url.QueryEscape(returnURL),
			Path:     "/",
			HttpOnly: true,
			Secure:   false, // Set to true in production with HTTPS
			SameSite: http.SameSiteLaxMode,
			MaxAge:   300, // 5 minutes
		})
	}
	
	http.Redirect(w, r, "/login", http.StatusFound)
}

// GetUserFromContext extracts user from request context
func GetUserFromContext(r *http.Request) *User {
	if user, ok := r.Context().Value("user").(*User); ok {
		return user
	}
	return nil
}
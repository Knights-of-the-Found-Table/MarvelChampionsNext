package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

func jsonDecode(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var cred credentials
	if err := jsonDecode(r, &cred); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(cred.Username) < 3 || len(cred.Password) < 6 {
		writeErr(w, http.StatusBadRequest, "username must be 3+ chars, password 6+ chars")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cred.Password), bcrypt.DefaultCost)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "hash failed")
		return
	}
	uid, err := s.Store.CreateUser(cred.Username, string(hash))
	if err != nil {
		writeErr(w, http.StatusConflict, "username taken")
		return
	}
	token, err := s.issueToken(uid)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "token failed")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"token": token, "username": cred.Username})
}

func (s *Server) handleAuthenticate(w http.ResponseWriter, r *http.Request) {
	var cred credentials
	if err := jsonDecode(r, &cred); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	user, err := s.Store.UserByName(cred.Username)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(cred.Password)) != nil {
		writeErr(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token, err := s.issueToken(user.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "token failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "username": user.Username})
}

func (s *Server) handleWhoami(w http.ResponseWriter, r *http.Request) {
	var id int64
	if _, err := fmt.Sscanf(userID(r), "%d", &id); err != nil {
		writeErr(w, http.StatusUnauthorized, "invalid user")
		return
	}
	user, err := s.Store.UserByID(id)
	if err != nil {
		writeErr(w, http.StatusUnauthorized, "unknown user")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": user.ID, "username": user.Username})
}

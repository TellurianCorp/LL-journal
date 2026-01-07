// LifeLogger LL-Journal
// https://api.lifelogger.life
// company: Tellurian Corp (https://www.telluriancorp.com)
// created in: December 2025

package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/telluriancorp/ll-journal/internal/journal"
)

type Handlers struct {
	service *journal.Service
}

func New(service *journal.Service) *Handlers {
	return &Handlers{service: service}
}

// getUserSub extracts user sub from request header (set by LL-proxy)
func getUserSub(r *http.Request) string {
	return r.Header.Get("X-User-Sub")
}

// Journal handlers

type CreateJournalRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

type UpdateJournalRequest struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

func (h *Handlers) CreateJournal(w http.ResponseWriter, r *http.Request) {
	userSub := getUserSub(r)
	if userSub == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateJournalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}

	journal, err := h.service.CreateJournal(r.Context(), userSub, req.Title, req.Description)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(journal)
}

func (h *Handlers) GetJournal(w http.ResponseWriter, r *http.Request) {
	userSub := getUserSub(r)
	if userSub == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	journalID := chi.URLParam(r, "id")
	journal, err := h.service.GetJournal(r.Context(), journalID, userSub)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Journal not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(journal)
}

func (h *Handlers) ListJournals(w http.ResponseWriter, r *http.Request) {
	userSub := getUserSub(r)
	if userSub == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	journals, err := h.service.ListJournals(r.Context(), userSub)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(journals)
}

func (h *Handlers) UpdateJournal(w http.ResponseWriter, r *http.Request) {
	userSub := getUserSub(r)
	if userSub == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	journalID := chi.URLParam(r, "id")
	var req UpdateJournalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get existing journal to preserve fields
	existing, err := h.service.GetJournal(r.Context(), journalID, userSub)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Journal not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Use existing values if not provided
	title := req.Title
	if title == "" {
		title = existing.Title
	}
	description := req.Description

	if err := h.service.UpdateJournal(r.Context(), journalID, userSub, title, description); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return updated journal
	updated, err := h.service.GetJournal(r.Context(), journalID, userSub)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (h *Handlers) DeleteJournal(w http.ResponseWriter, r *http.Request) {
	userSub := getUserSub(r)
	if userSub == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	journalID := chi.URLParam(r, "id")
	if err := h.service.DeleteJournal(r.Context(), journalID, userSub); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Journal not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Entry handlers

type CreateEntryRequest struct {
	EntryDate string `json:"entry_date"` // Format: YYYY-MM-DD
	Content   string `json:"content"`
}

type UpdateEntryRequest struct {
	Content string `json:"content"`
}

func (h *Handlers) CreateEntry(w http.ResponseWriter, r *http.Request) {
	userSub := getUserSub(r)
	if userSub == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	journalID := chi.URLParam(r, "journalId")
	var req CreateEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.EntryDate == "" {
		http.Error(w, "entry_date is required", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	entry, err := h.service.CreateEntry(r.Context(), userSub, journalID, req.EntryDate, req.Content)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(entry)
}

func (h *Handlers) GetEntry(w http.ResponseWriter, r *http.Request) {
	userSub := getUserSub(r)
	if userSub == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	journalID := chi.URLParam(r, "journalId")
	entryDate := chi.URLParam(r, "date")

	entry, content, err := h.service.GetEntry(r.Context(), userSub, journalID, entryDate)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Entry not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"entry":   entry,
		"content": string(content),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handlers) ListEntries(w http.ResponseWriter, r *http.Request) {
	userSub := getUserSub(r)
	if userSub == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	journalID := chi.URLParam(r, "journalId")
	entries, err := h.service.ListEntries(r.Context(), userSub, journalID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Journal not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func (h *Handlers) UpdateEntry(w http.ResponseWriter, r *http.Request) {
	userSub := getUserSub(r)
	if userSub == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	journalID := chi.URLParam(r, "journalId")
	entryDate := chi.URLParam(r, "date")

	var req UpdateEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Content == "" {
		http.Error(w, "content is required", http.StatusBadRequest)
		return
	}

	entry, err := h.service.UpdateEntry(r.Context(), userSub, journalID, entryDate, req.Content)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}

func (h *Handlers) DeleteEntry(w http.ResponseWriter, r *http.Request) {
	userSub := getUserSub(r)
	if userSub == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	journalID := chi.URLParam(r, "journalId")
	entryDate := chi.URLParam(r, "date")

	if err := h.service.DeleteEntry(r.Context(), userSub, journalID, entryDate); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Entry not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Version handlers

func (h *Handlers) ListVersions(w http.ResponseWriter, r *http.Request) {
	userSub := getUserSub(r)
	if userSub == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	journalID := chi.URLParam(r, "journalId")
	entryDate := chi.URLParam(r, "date")

	versions, err := h.service.ListVersions(r.Context(), userSub, journalID, entryDate)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Journal or entry not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	type VersionResponse struct {
		Hash        string `json:"hash"`
		Message     string `json:"message"`
		AuthorName  string `json:"author_name"`
		AuthorEmail string `json:"author_email"`
		CreatedAt   string `json:"created_at"`
	}

	response := make([]VersionResponse, len(versions))
	for i, v := range versions {
		response[i] = VersionResponse{
			Hash:        v.Hash,
			Message:     v.Message,
			AuthorName:  v.AuthorName,
			AuthorEmail: v.AuthorEmail,
			CreatedAt:   v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *Handlers) GetVersion(w http.ResponseWriter, r *http.Request) {
	userSub := getUserSub(r)
	if userSub == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	journalID := chi.URLParam(r, "journalId")
	entryDate := chi.URLParam(r, "date")
	commitHash := chi.URLParam(r, "commit")

	content, err := h.service.GetVersion(r.Context(), userSub, journalID, entryDate, commitHash)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Version not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"commit_hash": commitHash,
		"content":     string(content),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

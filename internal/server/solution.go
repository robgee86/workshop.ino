package server

import (
	"encoding/json"
	"net/http"

	"workshop.ino/internal/apps"
	"workshop.ino/internal/content"
)

// actionRequest identifies a step's attachment by the step it belongs to and its
// index within that step. The server re-derives every filesystem path from the
// trusted content model, so the browser never supplies a path directly.
type actionRequest struct {
	Step  string `json:"step"`
	Index int    `json:"index"`
}

type actionResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, resp actionResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// handleApplySolution applies step.Attachments[index] (which must be flagged a
// solution) over the step's target app folder, replacing its contents.
func (s *Server) handleApplySolution(w http.ResponseWriter, r *http.Request) {
	step, appDir, ok := s.resolveAction(w, r, func(st *content.Step, i int) (content.Attachment, bool) {
		if i < 0 || i >= len(st.Attachments) {
			return content.Attachment{}, false
		}
		a := st.Attachments[i]
		if !a.Solution {
			return content.Attachment{}, false
		}
		return a, true
	})
	if !ok {
		return
	}
	abs, err := s.resolveContentFile(step.FilePath, step.itemPath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, actionResponse{Error: "solution file not found"})
		return
	}
	if err := apps.RestoreArchive(appDir, abs); err != nil {
		writeJSON(w, http.StatusInternalServerError, actionResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, actionResponse{OK: true})
}

// resolvedStep carries the step plus the chosen item's content-relative path.
type resolvedStep struct {
	*content.Step
	itemPath string
}

// resolveAction decodes the request, looks up the step, picks the item with the
// given selector, and resolves the target app dir — writing the appropriate JSON
// error and returning ok=false on any failure.
func (s *Server) resolveAction(
	w http.ResponseWriter, r *http.Request,
	pick func(*content.Step, int) (content.Attachment, bool),
) (resolvedStep, string, bool) {
	var req actionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, actionResponse{Error: "invalid request"})
		return resolvedStep{}, "", false
	}
	ws, err := content.Scan(s.contentRoot)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, actionResponse{Error: "cannot read workshop content"})
		return resolvedStep{}, "", false
	}
	step := ws.FindStep(req.Step)
	if step == nil {
		writeJSON(w, http.StatusNotFound, actionResponse{Error: "unknown step"})
		return resolvedStep{}, "", false
	}
	item, valid := pick(step, req.Index)
	if !valid {
		writeJSON(w, http.StatusBadRequest, actionResponse{Error: "item not found for this step"})
		return resolvedStep{}, "", false
	}
	if step.App == "" {
		writeJSON(w, http.StatusBadRequest, actionResponse{Error: "no target app is configured for this step"})
		return resolvedStep{}, "", false
	}
	appDir, err := apps.Dir(s.appsRoot, step.App)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, actionResponse{Error: err.Error()})
		return resolvedStep{}, "", false
	}
	return resolvedStep{Step: step, itemPath: item.Path}, appDir, true
}

// resolveContentFile turns a frontmatter-relative path (against the step's .md
// directory) into a validated absolute path that stays within the content root.
func (s *Server) resolveContentFile(stepFile, relPath string) (string, error) {
	resolved, ok := content.ResolveRel(s.baseDir(stepFile), relPath)
	if !ok {
		return "", errInvalidPath
	}
	return resolveDownload(s.contentRoot, resolved)
}

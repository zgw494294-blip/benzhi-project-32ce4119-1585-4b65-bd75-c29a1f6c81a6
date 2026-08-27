package web

import (
	"net/http"
	"strings"

	"wetland-release-workbench/internal/domain"
	"wetland-release-workbench/internal/workflow"
)

func (s *Server) HandleJointRemediate(w http.ResponseWriter, r *http.Request) {
	var input domain.JointRemediationInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}
	view, err := s.service.JointRemediate(r.Context(), pathID(r, "id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) HandleRemediationQueue(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	from := query.Get("dueDateFrom")
	if from == "" {
		from = query.Get("from")
	}
	to := query.Get("dueDateTo")
	if to == "" {
		to = query.Get("to")
	}
	result, err := s.service.RemediationQueue(r.Context(), workflow.QueueFilters{Owner: query.Get("owner"), SpeciesCode: query.Get("speciesCode"), Severity: query.Get("severity"), Timing: query.Get("timing"), DueDateFrom: from, DueDateTo: to})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) HandleTrend(w http.ResponseWriter, r *http.Request) {
	result, err := s.service.Trend(r.Context(), pathID(r, "id"), pathID(r, "speciesCode"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) HandleTrendQuery(w http.ResponseWriter, r *http.Request) {
	result, err := s.service.Trend(r.Context(), pathID(r, "id"), strings.TrimSpace(r.URL.Query().Get("speciesCode")))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) HandleReviewManifest(w http.ResponseWriter, r *http.Request) {
	result, err := s.service.ReviewManifest(r.Context(), pathID(r, "id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) HandleRemediate(w http.ResponseWriter, r *http.Request) {
	var input domain.RemediationInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}
	view, err := s.service.Remediate(r.Context(), pathID(r, "id"), pathID(r, "issueID"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) HandleReview(w http.ResponseWriter, r *http.Request) {
	var input domain.ReviewInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}
	view, err := s.service.Review(r.Context(), pathID(r, "id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) HandleVerifyCredential(w http.ResponseWriter, r *http.Request) {
	result, err := s.service.VerifyCredential(r.Context(), pathID(r, "id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) HandleVerifySubmittedCredential(w http.ResponseWriter, r *http.Request) {
	var credential domain.ReleaseCredential
	if err := decodeJSON(r, &credential); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.service.VerifySubmittedCredential(r.Context(), credential)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

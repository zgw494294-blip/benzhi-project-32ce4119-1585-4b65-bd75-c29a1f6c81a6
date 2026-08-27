package web

import (
	"net/http"

	"wetland-release-workbench/internal/domain"
)

func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) HandleListBatches(w http.ResponseWriter, r *http.Request) {
	list, err := s.service.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) HandleCreateBatch(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateBatchInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}
	view, err := s.service.CreateBatch(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (s *Server) HandleGetBatch(w http.ResponseWriter, r *http.Request) {
	view, err := s.service.Get(r.Context(), pathID(r, "id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) HandleSubmitEvidence(w http.ResponseWriter, r *http.Request) {
	var input domain.EvidenceInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}
	view, err := s.service.SubmitEvidence(r.Context(), pathID(r, "id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) HandleSetPlan(w http.ResponseWriter, r *http.Request) {
	var input domain.PlanInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}
	view, err := s.service.SetPlan(r.Context(), pathID(r, "id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) HandleSubmitObservation(w http.ResponseWriter, r *http.Request) {
	var input domain.ObservationInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}
	view, err := s.service.SubmitObservation(r.Context(), pathID(r, "id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

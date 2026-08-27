package web

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"wetland-release-workbench/internal/domain"
	"wetland-release-workbench/internal/workflow"
)

type Server struct {
	service *workflow.Service
	mux     *http.ServeMux
}

func New(service *workflow.Service) http.Handler {
	s := &Server{service: service, mux: http.NewServeMux()}
	s.routes()
	return withMiddleware(s)
}

func (s *Server) routes() {
	staticFS, _ := fs.Sub(assets, "assets")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	s.mux.HandleFunc("GET /", s.HandleWorkbenchPage)
	s.mux.HandleFunc("GET /credentials/{id}", s.HandleCredentialPage)
	s.mux.HandleFunc("GET /api/health", s.HandleHealth)
	s.mux.HandleFunc("GET /api/batches", s.HandleListBatches)
	s.mux.HandleFunc("POST /api/batches", s.HandleCreateBatch)
	s.mux.HandleFunc("GET /api/batches/{id}", s.HandleGetBatch)
	s.mux.HandleFunc("POST /api/batches/{id}/evidence", s.HandleSubmitEvidence)
	s.mux.HandleFunc("POST /api/batches/{id}/plans", s.HandleSetPlan)
	s.mux.HandleFunc("POST /api/batches/{id}/observations", s.HandleSubmitObservation)
	s.mux.HandleFunc("POST /api/batches/{id}/issues/{issueID}/remediate", s.HandleRemediate)
	s.mux.HandleFunc("POST /api/batches/{id}/issues/joint-remediate", s.HandleJointRemediate)
	s.mux.HandleFunc("POST /api/batches/{id}/issues/bulk-remediate", s.HandleJointRemediate)
	s.mux.HandleFunc("POST /api/batches/{id}/issues/remediate", s.HandleJointRemediate)
	s.mux.HandleFunc("GET /api/remediation-queue", s.HandleRemediationQueue)
	s.mux.HandleFunc("GET /api/issues/queue", s.HandleRemediationQueue)
	s.mux.HandleFunc("GET /api/remediation/queue", s.HandleRemediationQueue)
	s.mux.HandleFunc("GET /api/batches/queue", s.HandleRemediationQueue)
	s.mux.HandleFunc("GET /api/batches/{id}/trends/{speciesCode}", s.HandleTrend)
	s.mux.HandleFunc("GET /api/batches/{id}/trend", s.HandleTrendQuery)
	s.mux.HandleFunc("GET /api/batches/{id}/observations/trend", s.HandleTrendQuery)
	s.mux.HandleFunc("GET /api/batches/{id}/species/{speciesCode}/trend", s.HandleTrend)
	s.mux.HandleFunc("GET /api/batches/{id}/review-manifest", s.HandleReviewManifest)
	s.mux.HandleFunc("GET /api/batches/{id}/manifest-preview", s.HandleReviewManifest)
	s.mux.HandleFunc("GET /api/batches/{id}/manifest", s.HandleReviewManifest)
	s.mux.HandleFunc("POST /api/batches/{id}/review", s.HandleReview)
	s.mux.HandleFunc("GET /api/credentials/{id}", s.HandleVerifyCredential)
	s.mux.HandleFunc("POST /api/credentials/verify", s.HandleVerifySubmittedCredential)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

type envelope struct {
	Data  any       `json:"data,omitempty"`
	Error *apiError `json:"error,omitempty"`
}
type apiError struct {
	Code    domain.ErrorCode  `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Data: value})
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := domain.ErrorCodeOf(err)
	switch code {
	case domain.CodeInvalid, domain.CodeEvidenceMissing, domain.CodeState:
		status = http.StatusUnprocessableEntity
	case domain.CodeNotFound:
		status = http.StatusNotFound
	case domain.CodeConflict, domain.CodeVersionConflict, domain.CodeIdempotency, domain.CodeFrozen:
		status = http.StatusConflict
	case domain.CodeForbidden:
		status = http.StatusForbidden
	}
	item := &apiError{Code: code, Message: "服务暂时无法完成请求"}
	var business *domain.BusinessError
	if errors.As(err, &business) {
		item.Message = business.Message
		item.Fields = business.Fields
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{Error: item})
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, (1<<20)+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return domain.NewError(domain.CodeInvalid, "JSON 请求体无效："+err.Error())
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return domain.NewError(domain.CodeInvalid, "请求体只能包含一个 JSON 对象")
	}
	return nil
}

func pathID(r *http.Request, name string) string { return strings.TrimSpace(r.PathValue(name)) }

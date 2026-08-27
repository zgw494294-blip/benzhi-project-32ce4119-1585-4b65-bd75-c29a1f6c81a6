package web

import (
	"context"
	"net/http/httptest"
	"testing"

	"wetland-release-workbench/internal/assessment"
	"wetland-release-workbench/internal/store"
	"wetland-release-workbench/internal/workflow"
)

func TestHTTPFullSelfCheck(t *testing.T) {
	persistence, err := store.Open("file:web-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer persistence.Close()
	service := workflow.New(persistence, assessment.NewSigner("test-key", []byte("test-http-secret")))
	server := httptest.NewServer(New(service))
	defer server.Close()
	if err := RunSelfCheck(context.Background(), server.URL); err != nil {
		t.Fatalf("HTTP 完整流程失败：%v", err)
	}
}

func TestWorkbenchAndSecurityHeaders(t *testing.T) {
	persistence, err := store.Open("file:page-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer persistence.Close()
	server := httptest.NewServer(New(workflow.New(persistence, assessment.NewSigner("test-key", []byte("test-http-secret")))))
	defer server.Close()
	response, err := server.Client().Get(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatalf("工作台状态码：%d", response.StatusCode)
	}
	if response.Header.Get("Content-Security-Policy") == "" || response.Header.Get("X-Request-ID") == "" {
		t.Fatal("缺少安全头或请求追踪 ID")
	}
}

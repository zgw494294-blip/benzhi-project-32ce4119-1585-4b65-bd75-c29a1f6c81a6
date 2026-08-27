package web

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"wetland-release-workbench/internal/domain"
	"wetland-release-workbench/internal/workflow"
)

type responseEnvelope struct {
	Data  json.RawMessage `json:"data"`
	Error *apiError       `json:"error"`
}

func RunSelfCheck(ctx context.Context, baseURL string) error {
	client := &http.Client{Timeout: 3 * time.Second}
	if err := waitReady(ctx, client, baseURL); err != nil {
		return err
	}
	today := time.Now().UTC().Format("2006-01-02")
	created := workflow.BatchView{}
	err := post(ctx, client, baseURL+"/api/batches", domain.CreateBatchInput{BatchCode: "SELF-CHECK-001", SiteCode: "WETLAND-A1", PlannedReleaseDate: today, Owner: "自检负责人", Actor: "苗圃交接员", IdempotencyKey: "selfcheck-create-001", Lots: []domain.SeedlingLot{{SpeciesCode: "SPARTINA_ALT", Quantity: 120}}}, &created)
	if err != nil {
		return fmt.Errorf("创建批次: %w", err)
	}
	id := created.Batch.ID
	evidence := workflow.BatchView{}
	err = post(ctx, client, baseURL+"/api/batches/"+id+"/evidence", domain.EvidenceInput{Meta: domain.CommandMeta{ExpectedVersion: created.Batch.Version, IdempotencyKey: "selfcheck-evidence-001", Actor: "苗圃交接员"}, SpeciesCode: "SPARTINA_ALT", NurseryOrigin: "东部生态苗圃", PermitDigest: "permit-sha256-selfcheck", QuarantineResult: "passed", HandoverAt: time.Now().UTC().Format(time.RFC3339), EvidenceRefs: []string{"handover-photo-001", "quarantine-note-001"}}, &evidence)
	if err != nil {
		return fmt.Errorf("登记证据: %w", err)
	}
	planned := workflow.BatchView{}
	err = post(ctx, client, baseURL+"/api/batches/"+id+"/plans", domain.PlanInput{Meta: domain.CommandMeta{ExpectedVersion: evidence.Batch.Version, IdempotencyKey: "selfcheck-plan-001", Actor: "生态技术员"}, Plan: domain.AcclimationPlan{SpeciesCode: "SPARTINA_ALT", ZoneCode: "ZONE-A01", DurationDays: 1, SalinityRange: domain.NumberRange{Min: 5, Max: 20}, WaterLevelRange: domain.NumberRange{Min: .1, Max: 1.5}, MinimumSurvivalRate: .8, MaximumDiseaseRate: .1, MinimumSampleCount: 10}}, &planned)
	if err != nil {
		return fmt.Errorf("配置方案: %w", err)
	}
	observed := workflow.BatchView{}
	err = post(ctx, client, baseURL+"/api/batches/"+id+"/observations", domain.ObservationInput{Meta: domain.CommandMeta{ExpectedVersion: planned.Batch.Version, IdempotencyKey: "selfcheck-observation-001", Actor: "生态技术员"}, Observation: domain.Observation{SpeciesCode: "SPARTINA_ALT", PlanRevision: 1, ObservedOn: today, ZoneCode: "ZONE-A01", SampledCount: 20, SurvivingCount: 19, DiseasedCount: 0, Salinity: 10, WaterLevel: .8, Notes: "自检观察正常"}}, &observed)
	if err != nil {
		return fmt.Errorf("提交观察: %w", err)
	}
	if observed.Batch.Status != domain.StatusReviewReady {
		return fmt.Errorf("观察后状态应为 review_ready，实际为 %s", observed.Batch.Status)
	}
	var preview workflow.ManifestPreview
	if err = get(ctx, client, baseURL+"/api/batches/"+id+"/review-manifest", &preview); err != nil {
		return fmt.Errorf("读取冻结清单预览: %w", err)
	}
	approved := workflow.BatchView{}
	err = post(ctx, client, baseURL+"/api/batches/"+id+"/review", domain.ReviewInput{Meta: domain.CommandMeta{ExpectedVersion: observed.Batch.Version, IdempotencyKey: "selfcheck-review-001", Actor: "独立复核负责人"}, Decision: "approve", ConfirmedManifestDigest: preview.Digest}, &approved)
	if err != nil {
		return fmt.Errorf("批准批次: %w", err)
	}
	if approved.Credential == nil {
		return fmt.Errorf("批准后未签发凭据")
	}
	var verified workflow.CredentialVerification
	if err = get(ctx, client, baseURL+"/api/credentials/"+approved.Credential.ID, &verified); err != nil {
		return fmt.Errorf("验证凭据: %w", err)
	}
	if !verified.Valid {
		return fmt.Errorf("凭据签名验证失败")
	}
	var submitted workflow.CredentialVerification
	if err = post(ctx, client, baseURL+"/api/credentials/verify", *approved.Credential, &submitted); err != nil {
		return fmt.Errorf("提交凭据验证: %w", err)
	}
	if !submitted.Valid {
		return fmt.Errorf("提交的完整凭据验证失败")
	}
	return nil
}

func waitReady(ctx context.Context, client *http.Client, baseURL string) error {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待服务就绪: %w", ctx.Err())
		case <-ticker.C:
			var health map[string]string
			if err := get(ctx, client, baseURL+"/api/health", &health); err == nil && health["status"] == "ok" {
				return nil
			}
		}
	}
}

func post(ctx context.Context, client *http.Client, url string, input, target any) error {
	raw, _ := json.Marshal(input)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return do(client, req, target)
}
func get(ctx context.Context, client *http.Client, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	return do(client, req, target)
}
func do(client *http.Client, req *http.Request, target any) error {
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var envelope responseEnvelope
	if err = json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if envelope.Error != nil {
			return fmt.Errorf("%s: %s", envelope.Error.Code, envelope.Error.Message)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return json.Unmarshal(envelope.Data, target)
}

package workflow

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"wetland-release-workbench/internal/domain"
)

// VerifySubmittedCredential 校验访问者提交的完整凭据，不依赖数据库中的签名字段。
// 数据库查询只用于在签名通过后确认该凭据仍对应本服务冻结的批准事实。
func (s *Service) VerifySubmittedCredential(ctx context.Context, submitted domain.ReleaseCredential) (CredentialVerification, error) {
	if submitted.ID == "" || submitted.BatchID == "" || submitted.ManifestDigest == "" || submitted.Signature == "" {
		return CredentialVerification{}, domain.NewError(domain.CodeInvalid, "提交的凭据缺少 ID、批次、摘要或签名")
	}
	if !s.signer.Verify(submitted) {
		return CredentialVerification{Valid: false, Credential: submitted, Message: "凭据内容或签名已被修改，请勿投放"}, nil
	}
	stored, err := s.store.Credential(ctx, submitted.ID)
	if err != nil {
		return CredentialVerification{}, err
	}
	if credentialFingerprint(stored) != credentialFingerprint(submitted) {
		return CredentialVerification{Valid: false, Credential: submitted, Message: "凭据与已冻结批准事实不一致，请勿投放"}, nil
	}
	return CredentialVerification{Valid: true, Credential: submitted, Message: "凭据签名完整，且与冻结清单中的批准事实一致"}, nil
}

func credentialFingerprint(credential domain.ReleaseCredential) string {
	copy := credential
	copy.Signature = ""
	raw, _ := json.Marshal(copy)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

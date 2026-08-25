package contracts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
)

const CapabilityMasteryBundleContractVersion = "capability-mastery-bundle.v1"

// CapabilityMasteryContractError is returned for both unsupported versions and
// invalid joins. Callers must not continue with a partially decoded bundle.
type CapabilityMasteryContractError struct {
	Code            string
	ContractVersion string
	Detail          string
}

func (e *CapabilityMasteryContractError) Error() string {
	return fmt.Sprintf("capability mastery contract %s: %s", e.Code, e.Detail)
}

var bundleSHA256 = regexp.MustCompile(`^[a-f0-9]{64}$`)

// ValidateCapabilityMasteryBundleJSON validates the additive v1 bundle at the
// Runtime boundary. Runtime reads identities and provenance; it never owns
// question prose, learner progress, or a mastery verdict.
func ValidateCapabilityMasteryBundleJSON(payload []byte) error {
	var root map[string]any
	if err := json.Unmarshal(payload, &root); err != nil {
		return &CapabilityMasteryContractError{Code: "invalid_bundle", Detail: "invalid JSON"}
	}
	version, _ := root["contractVersion"].(string)
	if version != CapabilityMasteryBundleContractVersion {
		return &CapabilityMasteryContractError{Code: "unsupported_contract_version", ContractVersion: version, Detail: "expected capability-mastery-bundle.v1"}
	}
	for section, expected := range map[string]string{
		"capabilityDossier": "capability-dossier.v1", "assessmentPlan": "capability-assessment-plan.v1",
		"learningSession": "learning-session.v1", "evidence": "capability-evidence.v2",
		"mastery": "capability-mastery.v2", "reflection": "session-reflection.v1",
	} {
		candidate, ok := object(root[section])
		if !ok || candidate["contractVersion"] != expected {
			return bundleError("invalid_bundle", fmt.Sprintf("%s has unsupported or missing contract version", section))
		}
	}
	dossier, _ := object(root["capabilityDossier"])
	plan, _ := object(root["assessmentPlan"])
	session, _ := object(root["learningSession"])
	mastery, _ := object(root["mastery"])
	primary, ok := object(dossier["primaryQuestion"])
	if !ok || primary["role"] != "primary" || primary["status"] != "published" || !localesContain(primary["locales"], "en", "ru") || !validHash(primary["contentHash"]) {
		return bundleError("incomplete-role-specific-primary-card", "primary QuestionCard is not a published en/ru card")
	}
	if dossier["capabilityKey"] == "" || objectValue(dossier, "registry", "key") != dossier["capabilityKey"] || objectValue(dossier, "registry", "lifecycle") != "active" {
		return bundleError("inactive-capability-registry", "capability registry identity is not active")
	}
	family, ok := object(dossier["taskFamily"])
	if !ok || family["assessmentPlanId"] != plan["planId"] {
		return bundleError("family-assessment-plan-missing", "released TaskFamily has no matching assessment plan")
	}
	gates, ok := plan["gates"].([]any)
	if !ok || len(gates) != 5 {
		return bundleError("family-assessment-plan-missing", "assessment plan must contain five gates")
	}
	selected, ok := object(session["taskRevision"])
	if !ok || !validHash(selected["immutableHash"]) || session["cardRevisionId"] != primary["revisionId"] || session["cardContentHash"] != primary["contentHash"] {
		return bundleError("stale-question-card-hash", "session does not carry the current QuestionCard identity")
	}
	revisions, ok := dossier["revisions"].([]any)
	if !ok {
		return bundleError("stale-task-revision-hash", "dossier has no task revisions")
	}
	var matching map[string]any
	for _, candidate := range revisions {
		item, ok := object(candidate)
		if ok && item["taskId"] == selected["taskId"] && number(item["revision"]) == number(selected["revision"]) {
			matching = item
			break
		}
	}
	if matching == nil || matching["immutableHash"] != selected["immutableHash"] {
		return bundleError("stale-task-revision-hash", "session does not carry the current TaskRevision identity")
	}
	if matching["profile"] != selected["profile"] || !validProfile(selected["profile"]) {
		return bundleError("wrong-profile-for-capability", "selected profile is incompatible with the task revision")
	}
	if session["releaseReadiness"] == "released" && (family["status"] != "released" || family["runnable"] != true || matching["status"] != "released") {
		return bundleError("contradictory-released-runnable", "released session is not backed by a released runnable revision")
	}
	if mastery["provenance"] != "human" {
		return bundleError("non-human-mastery-provenance", "mastery provenance must be human")
	}
	return nil
}

func bundleError(code, detail string) error {
	return &CapabilityMasteryContractError{Code: code, ContractVersion: CapabilityMasteryBundleContractVersion, Detail: detail}
}
func object(value any) (map[string]any, bool) { item, ok := value.(map[string]any); return item, ok }
func objectValue(root map[string]any, path ...string) any {
	var current any = root
	for _, key := range path {
		item, ok := object(current)
		if !ok {
			return nil
		}
		current = item[key]
	}
	return current
}
func number(value any) float64 { n, _ := value.(float64); return n }
func validHash(value any) bool {
	text, ok := value.(string)
	return ok && bundleSHA256.MatchString(text)
}
func validProfile(value any) bool {
	text, ok := value.(string)
	return ok && (text == "go" || text == "java" || text == "dotnet" || text == "node" || text == "postgres")
}
func localesContain(value any, required ...string) bool {
	values, ok := value.([]any)
	if !ok {
		return false
	}
	found := map[string]bool{}
	for _, value := range values {
		if text, ok := value.(string); ok {
			found[text] = true
		}
	}
	for _, want := range required {
		if !found[want] {
			return false
		}
	}
	return true
}

// Keep crypto/sha256 and hex linked in generated docs that consume this file;
// the helper is also useful to callers that need a deterministic fixture hash.
func CapabilityMasteryFixtureDigest(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

//go:build integration

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	hypershiftv1beta1 "github.com/openshift/hypershift/api/hypershift/v1beta1"
	hyperfleetv1alpha1 "github.com/typeid/hyperfleet-operator/api/v1alpha1"

	"github.com/openshift/rosa-regional-platform-api/pkg/clients/hyperfleetdb"
	"github.com/openshift/rosa-regional-platform-api/pkg/middleware"
	"github.com/openshift/rosa-regional-platform-api/pkg/types"
)

const testAccountID = "123456789012"

func newTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	_ = hyperfleetv1alpha1.AddToScheme(s)
	return s
}

func testContext(accountID string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, middleware.ContextKeyAccountID, accountID)
	ctx = context.WithValue(ctx, middleware.ContextKeyCallerARN, "arn:aws:iam::"+accountID+":user/test")
	return ctx
}

// testClusterCR creates a cluster CR with Namespace=clusterID (UUID),
// Name=clusterName (human-readable), labeled with accountID.
func testClusterCR(clusterID, clusterName, accountID string) *hyperfleetv1alpha1.Cluster {
	return &hyperfleetv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: clusterID,
			Labels:    map[string]string{"hyperfleet.io/account-id": accountID},
		},
		Spec: hyperfleetv1alpha1.ClusterSpec{
			HostedCluster: hypershiftv1beta1.HostedClusterSpec{
				Platform: hypershiftv1beta1.PlatformSpec{
					Type: hypershiftv1beta1.AWSPlatform,
					AWS: &hypershiftv1beta1.AWSPlatformSpec{
						Region: "us-east-1",
					},
				},
			},
		},
	}
}

func TestClusterHandler_List_Success(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		testClusterCR("uuid-1", "cluster-1", testAccountID),
		testClusterCR("uuid-2", "cluster-2", testAccountID),
	).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/clusters", nil)
	req = req.WithContext(testContext(testAccountID))

	w := httptest.NewRecorder()
	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&result)

	if int(result["total"].(float64)) != 2 {
		t.Errorf("expected total=2, got %v", result["total"])
	}
	items := result["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestClusterHandler_List_Empty(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/clusters", nil)
	req = req.WithContext(testContext(testAccountID))

	w := httptest.NewRecorder()
	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&result)

	if int(result["total"].(float64)) != 0 {
		t.Errorf("expected total=0, got %v", result["total"])
	}
}

func TestClusterHandler_List_Pagination(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		testClusterCR("uuid-c1", "c1", testAccountID),
		testClusterCR("uuid-c2", "c2", testAccountID),
		testClusterCR("uuid-c3", "c3", testAccountID),
	).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/clusters?limit=2&offset=1", nil)
	req = req.WithContext(testContext(testAccountID))

	w := httptest.NewRecorder()
	handler.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&result)

	if int(result["total"].(float64)) != 3 {
		t.Errorf("expected total=3, got %v", result["total"])
	}
	items := result["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("expected 2 items (offset=1, limit=2 of 3), got %d", len(items))
	}
}

func TestClusterHandler_Create_Success(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "my-cluster",
		"spec": map[string]interface{}{
			"platform": map[string]interface{}{
				"aws": map[string]interface{}{
					"region": "us-east-1",
				},
			},
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v0/clusters", bytes.NewReader(body))
	req = req.WithContext(testContext(testAccountID))

	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&result)

	if result["id"] == nil || result["id"] == "" {
		t.Error("expected non-empty cluster ID")
	}
	if result["name"] != "my-cluster" {
		t.Errorf("expected name=my-cluster, got %v", result["name"])
	}
}

func TestClusterHandler_Create_SetsCreatorARN(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "my-cluster",
		"spec": map[string]interface{}{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v0/clusters", bytes.NewReader(body))
	req = req.WithContext(testContext(testAccountID))

	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&result)

	if result["created_by"] != "arn:aws:iam::"+testAccountID+":user/test" {
		t.Errorf("expected creatorARN in created_by, got %v", result["created_by"])
	}
}

func TestClusterHandler_Create_InvalidJSON(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	req := httptest.NewRequest(http.MethodPost, "/api/v0/clusters", bytes.NewReader([]byte("not json")))
	req = req.WithContext(testContext(testAccountID))

	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestClusterHandler_Create_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		body map[string]interface{}
	}{
		{"missing name", map[string]interface{}{"spec": map[string]interface{}{}}},
		{"missing spec", map[string]interface{}{"name": "test"}},
		{"empty name", map[string]interface{}{"name": "", "spec": map[string]interface{}{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := newTestScheme()
			fc := fake.NewClientBuilder().WithScheme(scheme).Build()
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v0/clusters", bytes.NewReader(body))
			req = req.WithContext(testContext(testAccountID))

			w := httptest.NewRecorder()
			handler.Create(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestClusterHandler_Get_Success(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		testClusterCR("cluster-123", "test-cluster", testAccountID),
	).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/clusters/cluster-123", nil)
	req = req.WithContext(testContext(testAccountID))
	req = mux.SetURLVars(req, map[string]string{"id": "cluster-123"})

	w := httptest.NewRecorder()
	handler.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&result)

	if result["id"] != "cluster-123" {
		t.Errorf("expected id=cluster-123, got %v", result["id"])
	}
	if result["name"] != "test-cluster" {
		t.Errorf("expected name=test-cluster, got %v", result["name"])
	}
}

func TestClusterHandler_Get_NotFound(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/clusters/no-such-cluster", nil)
	req = req.WithContext(testContext(testAccountID))
	req = mux.SetURLVars(req, map[string]string{"id": "no-such-cluster"})

	w := httptest.NewRecorder()
	handler.Get(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	var errResp map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&errResp)
	if errResp["code"] != "CLUSTERS-MGMT-GET-001" {
		t.Errorf("expected code CLUSTERS-MGMT-GET-001, got %v", errResp["code"])
	}
}

func TestClusterHandler_Delete_Success(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		testClusterCR("cluster-123", "test-cluster", testAccountID),
	).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	req := httptest.NewRequest(http.MethodDelete, "/api/v0/clusters/cluster-123", nil)
	req = req.WithContext(testContext(testAccountID))
	req = mux.SetURLVars(req, map[string]string{"id": "cluster-123"})

	w := httptest.NewRecorder()
	handler.Delete(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&result)
	if result["cluster_id"] != "cluster-123" {
		t.Errorf("expected cluster_id=cluster-123, got %v", result["cluster_id"])
	}
}

func TestClusterHandler_Delete_NotFound(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	req := httptest.NewRequest(http.MethodDelete, "/api/v0/clusters/no-such-cluster", nil)
	req = req.WithContext(testContext(testAccountID))
	req = mux.SetURLVars(req, map[string]string{"id": "no-such-cluster"})

	w := httptest.NewRecorder()
	handler.Delete(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestClusterHandler_GetStatus_Success(t *testing.T) {
	cr := testClusterCR("cluster-123", "test-cluster", testAccountID)
	cr.Status = hyperfleetv1alpha1.ClusterStatus{
		ObservedGeneration: 1,
		Phase:              "Ready",
		Conditions: []metav1.Condition{
			{
				Type:   "Ready",
				Status: metav1.ConditionTrue,
				Reason: "ClusterReady",
			},
		},
	}

	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr).
		WithStatusSubresource(cr).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/clusters/cluster-123/statuses", nil)
	req = req.WithContext(testContext(testAccountID))
	req = mux.SetURLVars(req, map[string]string{"id": "cluster-123"})

	w := httptest.NewRecorder()
	handler.GetStatus(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&result)

	if result["cluster_id"] != "cluster-123" {
		t.Errorf("expected cluster_id=cluster-123, got %v", result["cluster_id"])
	}
}

func TestClusterHandler_GetStatus_NotFound(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/clusters/no-such/statuses", nil)
	req = req.WithContext(testContext(testAccountID))
	req = mux.SetURLVars(req, map[string]string{"id": "no-such"})

	w := httptest.NewRecorder()
	handler.GetStatus(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestClusterHandler_Update_Success(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		testClusterCR("cluster-123", "test-cluster", testAccountID),
	).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	body, _ := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{
			"name": "updated-name",
		},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v0/clusters/cluster-123", bytes.NewReader(body))
	req = req.WithContext(testContext(testAccountID))
	req = mux.SetURLVars(req, map[string]string{"id": "cluster-123"})

	w := httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&result)

	if result["name"] != "updated-name" {
		t.Errorf("expected name=updated-name, got %v", result["name"])
	}
}

func TestClusterHandler_Update_NotFound(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	body, _ := json.Marshal(map[string]interface{}{
		"spec": map[string]interface{}{"name": "x"},
	})

	req := httptest.NewRequest(http.MethodPut, "/api/v0/clusters/no-such", bytes.NewReader(body))
	req = req.WithContext(testContext(testAccountID))
	req = mux.SetURLVars(req, map[string]string{"id": "no-such"})

	w := httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestClusterHandler_Update_MissingSpec(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	body, _ := json.Marshal(map[string]interface{}{})

	req := httptest.NewRequest(http.MethodPut, "/api/v0/clusters/cluster-123", bytes.NewReader(body))
	req = req.WithContext(testContext(testAccountID))
	req = mux.SetURLVars(req, map[string]string{"id": "cluster-123"})

	w := httptest.NewRecorder()
	handler.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestClusterHandler_Create_DuplicateName(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		testClusterCR("existing-id", "test-cluster", testAccountID),
	).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "test-cluster",
		"spec": map[string]interface{}{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v0/clusters", bytes.NewReader(body))
	req = req.WithContext(testContext(testAccountID))

	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 for duplicate name, got %d: %s", w.Code, w.Body.String())
	}

	var errResp map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&errResp)
	if errResp["code"] != "CLUSTERS-MGMT-CREATE-005" {
		t.Errorf("expected code CLUSTERS-MGMT-CREATE-005, got %v", errResp["code"])
	}
}

func TestClusterHandler_Create_SameNameDifferentAccount(t *testing.T) {
	otherAccount := "999999999999"
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		testClusterCR("existing-id", "test-cluster", otherAccount),
	).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	body, _ := json.Marshal(map[string]interface{}{
		"name": "test-cluster",
		"spec": map[string]interface{}{},
	})

	req := httptest.NewRequest(http.MethodPost, "/api/v0/clusters", bytes.NewReader(body))
	req = req.WithContext(testContext(testAccountID))

	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 (same name in different account is allowed), got %d: %s", w.Code, w.Body.String())
	}
}

// TestClusterHandler_Create_ValidationRejectsServiceSetField tests that the validator
// rejects service-set fields (like creatorARN) in the create request spec.
// On pgruntime, specs are strongly-typed so only fields present on ClusterSpec
// can be tested here (creatorARN is service-set and exists on the struct).
func TestClusterHandler_Create_ValidationRejectsServiceSetField(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	fv := middleware.NewFieldValidator()
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", fv, logger)

	reqBody := types.ClusterCreateRequest{
		Name: "test-cluster",
		Spec: &hyperfleetv1alpha1.ClusterSpec{
			CreatorARN: "arn:aws:iam::123456789012:user/test",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v0/clusters", bytes.NewReader(body))
	req = req.WithContext(testContext(testAccountID))

	w := httptest.NewRecorder()
	handler.Create(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d (body: %s)", w.Code, w.Body.String())
	}

	var errorResp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&errorResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errorResp["code"] != "CLUSTERS-MGMT-VALIDATE-001" {
		t.Errorf("expected code CLUSTERS-MGMT-VALIDATE-001, got %v", errorResp["code"])
	}

	details, ok := errorResp["details"].([]interface{})
	if !ok || len(details) == 0 {
		t.Fatal("expected details with at least one validation error")
	}
	found := false
	for _, d := range details {
		if dm, ok := d.(map[string]interface{}); ok && dm["field"] == "spec.creatorARN" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected validation error for spec.creatorARN, got %v", details)
	}
}

// TestClusterHandler_Create_NilValidatorBypasses verifies that when fieldValidator
// is nil, validation is skipped and the request proceeds to the normal create flow.
func TestClusterHandler_Create_NilValidatorBypasses(t *testing.T) {
	scheme := newTestScheme()
	fc := fake.NewClientBuilder().WithScheme(scheme).Build()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := NewClusterHandler(hyperfleetdb.NewClientFrom(fc, logger), "https://oidc.example.com", nil, logger)

	reqBody := types.ClusterCreateRequest{
		Name: "test-cluster",
		Spec: &hyperfleetv1alpha1.ClusterSpec{
			CreatorARN: "arn:aws:iam::123456789012:user/test",
		},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v0/clusters", bytes.NewReader(body))
	req = req.WithContext(testContext(testAccountID))

	w := httptest.NewRecorder()
	handler.Create(w, req)

	// With nil validator, the request should NOT get 422 — it should proceed
	// past validation to the create flow (which succeeds or fails for other reasons)
	if w.Code == http.StatusUnprocessableEntity {
		t.Errorf("expected validation to be bypassed with nil validator, but got 422")
	}
}

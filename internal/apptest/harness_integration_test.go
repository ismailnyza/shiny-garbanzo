//go:build integration

package apptest

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/google/uuid"
	gorilla "github.com/gorilla/websocket"
	"github.com/ismael/qr-restaurant/internal/app"
	"github.com/ismael/qr-restaurant/internal/shared/config"
	"github.com/ismael/qr-restaurant/internal/shared/database"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type testServer struct {
	t      *testing.T
	router http.Handler
	schema string
	db     *gorm.DB
	admin  string
}

type envelope struct {
	Success   bool            `json:"success"`
	RequestID string          `json:"request_id"`
	Data      json.RawMessage `json:"data"`
	Error     *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	baseURL := os.Getenv("QR_TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("set QR_TEST_DATABASE_URL to run integration tests")
	}

	schema := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	adminDB, err := sql.Open("postgres", baseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = adminDB.Close() })

	if _, err := adminDB.Exec(`CREATE SCHEMA ` + schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminDB.Exec(`DROP SCHEMA IF EXISTS ` + schema + ` CASCADE`)
	})

	dbURL := withSearchPath(t, baseURL, schema)
	if err := database.RunMigrations(dbURL, "file://../../migrations"); err != nil {
		t.Fatalf("migrate test schema: %v", err)
	}

	gormDB, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect gorm: %v", err)
	}

	cfg := &config.Config{
		AppEnv:            "test",
		Port:              "0",
		FrontendBaseURL:   "https://app.example.test",
		JWTSecret:         "0123456789abcdef0123456789abcdef",
		JWTExpiryHours:    1,
		BCryptCost:        4,
		SessionTTLHours:   8,
		RateLimitPublic:   1000,
		RateLimitAdmin:    1000,
		R2BucketName:      "test",
		MigratePath:       "file://../../migrations",
		R2PublicURL:       "https://assets.example.test",
		R2AccountID:       "",
		R2AccessKeyID:     "",
		R2SecretAccessKey: "",
	}

	return &testServer{
		t:      t,
		router: app.NewRouter(cfg, gormDB, nil),
		schema: schema,
		db:     gormDB,
		admin:  baseURL,
	}
}

func TestRequestIDAndMetricsEndpoint(t *testing.T) {
	s := newTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("X-Request-ID", "pilot-request-1")
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected health 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-Request-ID"); got != "pilot-request-1" {
		t.Fatalf("expected request id header to round-trip, got %q", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec = httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized, got %d: %s", rec.Code, rec.Body.String())
	}
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if env.RequestID == "" {
		t.Fatal("expected generated request id in error envelope")
	}

	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec = httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected metrics 200, got %d: %s", rec.Code, rec.Body.String())
	}
	for _, metric := range []string{
		"http_requests_total",
		"http_responses_total",
		"order_creation_success_total",
		"websocket_active_connections",
		"db_open_connections",
	} {
		if !strings.Contains(rec.Body.String(), metric) {
			t.Fatalf("expected metric %q in output: %s", metric, rec.Body.String())
		}
	}
}

func withSearchPath(t *testing.T, raw, schema string) string {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse QR_TEST_DATABASE_URL: %v", err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	return u.String()
}

func (s *testServer) request(method, path string, token string, body any, want int, out any) {
	s.t.Helper()
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		b, err := json.Marshal(body)
		if err != nil {
			s.t.Fatal(err)
		}
		reader = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	if rec.Code != want {
		s.t.Fatalf("%s %s: expected %d, got %d: %s", method, path, want, rec.Code, rec.Body.String())
	}
	if out == nil || rec.Code == http.StatusNoContent {
		return
	}

	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		s.t.Fatalf("decode envelope: %v: %s", err, rec.Body.String())
	}
	if !env.Success {
		s.t.Fatalf("unexpected error envelope: %+v", env.Error)
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		s.t.Fatalf("decode data: %v: %s", err, env.Data)
	}
}

func (s *testServer) registerAndLogin(email string) string {
	s.request(http.MethodPost, "/api/v1/auth/register", "", map[string]any{
		"email":    email,
		"password": "password123",
		"role":     "OWNER",
	}, http.StatusCreated, &map[string]any{})

	var login struct {
		AccessToken string `json:"access_token"`
	}
	s.request(http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email":    email,
		"password": "password123",
	}, http.StatusOK, &login)
	return login.AccessToken
}

func (s *testServer) createTenant(token, name string) (restaurantID, tableID, qrToken, categoryID, itemID string) {
	var restaurant struct {
		ID string `json:"id"`
	}
	s.request(http.MethodPost, "/api/v1/restaurants", token, map[string]any{
		"name": name,
	}, http.StatusCreated, &restaurant)

	var table struct {
		ID      string `json:"id"`
		QRToken string `json:"qr_token"`
	}
	s.request(http.MethodPost, fmt.Sprintf("/api/v1/restaurants/%s/tables", restaurant.ID), token, map[string]any{
		"table_code": "T1",
	}, http.StatusCreated, &table)

	var category struct {
		ID string `json:"id"`
	}
	s.request(http.MethodPost, fmt.Sprintf("/api/v1/restaurants/%s/menu/categories", restaurant.ID), token, map[string]any{
		"name": "Mains",
	}, http.StatusCreated, &category)

	var item struct {
		ID string `json:"id"`
	}
	s.request(http.MethodPost, fmt.Sprintf("/api/v1/restaurants/%s/menu/categories/%s/items", restaurant.ID, category.ID), token, map[string]any{
		"name":        "Rice",
		"price_cents": 1200,
	}, http.StatusCreated, &item)

	return restaurant.ID, table.ID, table.QRToken, category.ID, item.ID
}

func (s *testServer) createModifierTenant(token, name string) (restaurantID, tableID, qrToken, categoryID, itemID string) {
	restaurantID, tableID, qrToken, categoryID, _ = s.createTenant(token, name)

	var item struct {
		ID string `json:"id"`
	}
	s.request(http.MethodPost, fmt.Sprintf("/api/v1/restaurants/%s/menu/categories/%s/items", restaurantID, categoryID), token, map[string]any{
		"name":        "Burger",
		"price_cents": 1000,
		"options_config": map[string]any{
			"modifier_groups": []map[string]any{
				{
					"id":                 "size",
					"name":               "Choose 1 size",
					"required":           true,
					"min_selected":       1,
					"max_selected":       1,
					"selection_strategy": "SINGLE_SELECT",
					"active":             true,
					"options": []map[string]any{
						{"id": "regular", "name": "Regular", "price_delta": 0, "active": true, "maximum_quantity_per_option": 1},
						{"id": "large", "name": "Large", "price_delta": 200, "active": true, "maximum_quantity_per_option": 1},
					},
				},
				{
					"id":                        "toppings",
					"name":                      "Choose up to 3 toppings",
					"min_selected":              0,
					"max_selected":              3,
					"selection_strategy":        "MULTI_SELECT",
					"allow_multiple_quantities": true,
					"active":                    true,
					"options": []map[string]any{
						{"id": "cheese", "name": "Extra cheese", "price_delta": 200, "active": true, "maximum_quantity_per_option": 2},
						{"id": "bacon", "name": "Bacon", "price_delta": 300, "active": true, "maximum_quantity_per_option": 1},
						{"id": "old", "name": "Inactive", "price_delta": 100, "active": false, "maximum_quantity_per_option": 1},
					},
				},
			},
		},
	}, http.StatusCreated, &item)
	return restaurantID, tableID, qrToken, categoryID, item.ID
}

func (s *testServer) scan(qrToken string) string {
	var scan struct {
		SessionToken string `json:"session_token"`
	}
	s.request(http.MethodGet, "/api/v1/public/scan?token="+url.QueryEscape(qrToken), "", nil, http.StatusOK, &scan)
	return scan.SessionToken
}

func TestFullAPIFlowAndIdempotencyReplay(t *testing.T) {
	s := newTestServer(t)
	token := s.registerAndLogin("owner-" + uuid.NewString() + "@example.test")
	_, _, qrToken, _, itemID := s.createTenant(token, "Test Cafe")
	sessionToken := s.scan(qrToken)

	key := uuid.NewString()
	body := map[string]any{
		"idempotency_key": key,
		"items": []map[string]any{{
			"menu_item_id": itemID,
			"quantity":     2,
		}},
	}
	var first, second struct {
		ID         string `json:"id"`
		TotalCents int    `json:"total_cents"`
	}
	s.request(http.MethodPost, "/api/v1/public/sessions/"+sessionToken+"/orders", "", body, http.StatusCreated, &first)
	s.request(http.MethodPost, "/api/v1/public/sessions/"+sessionToken+"/orders", "", body, http.StatusCreated, &second)

	if first.ID != second.ID {
		t.Fatalf("expected idempotent replay to return same order, got %s and %s", first.ID, second.ID)
	}
	if first.TotalCents != 2400 {
		t.Fatalf("expected total 2400, got %d", first.TotalCents)
	}
}

func TestModifierPricingValidationAndSnapshotImmutability(t *testing.T) {
	s := newTestServer(t)
	token := s.registerAndLogin("owner-" + uuid.NewString() + "@example.test")
	restaurantID, _, qrToken, _, itemID := s.createModifierTenant(token, "Modifier Cafe")
	sessionToken := s.scan(qrToken)

	validBody := map[string]any{
		"items": []map[string]any{{
			"menu_item_id": itemID,
			"quantity":     2,
			"selected_options": map[string]any{
				"modifier_selections": []map[string]any{
					{"group_id": "size", "options": []map[string]any{{"option_id": "large", "quantity": 1}}},
					{"group_id": "toppings", "options": []map[string]any{
						{"option_id": "cheese", "quantity": 2},
						{"option_id": "bacon", "quantity": 1},
					}},
				},
			},
		}},
	}
	var orderResp struct {
		ID         string `json:"id"`
		TotalCents int    `json:"total_cents"`
		Items      []struct {
			SelectedOptions map[string]any `json:"selected_options"`
		} `json:"items"`
	}
	s.request(http.MethodPost, "/api/v1/public/sessions/"+sessionToken+"/orders", "", validBody, http.StatusCreated, &orderResp)
	if orderResp.TotalCents != 3800 {
		t.Fatalf("expected total 3800, got %d", orderResp.TotalCents)
	}
	if len(orderResp.Items) != 1 || orderResp.Items[0].SelectedOptions["modifier_groups"] == nil {
		t.Fatalf("expected immutable modifier snapshot, got %#v", orderResp.Items)
	}

	invalidBody := map[string]any{
		"items": []map[string]any{{
			"menu_item_id": itemID,
			"quantity":     1,
			"selected_options": map[string]any{
				"modifier_selections": []map[string]any{
					{"group_id": "toppings", "options": []map[string]any{{"option_id": "cheese", "quantity": 3}}},
				},
			},
		}},
	}
	s.request(http.MethodPost, "/api/v1/public/sessions/"+sessionToken+"/orders", "", invalidBody, http.StatusBadRequest, nil)

	var item struct {
		ID string `json:"id"`
	}
	s.request(http.MethodPatch, fmt.Sprintf("/api/v1/restaurants/%s/menu/items/%s", restaurantID, itemID), token, map[string]any{
		"price_cents": 9999,
	}, http.StatusOK, &item)

	var fetched struct {
		ID         string `json:"id"`
		TotalCents int    `json:"total_cents"`
	}
	s.request(http.MethodGet, fmt.Sprintf("/api/v1/restaurants/%s/orders/%s", restaurantID, orderResp.ID), token, nil, http.StatusOK, &fetched)
	if fetched.TotalCents != 3800 {
		t.Fatalf("expected historical order total to remain 3800, got %d", fetched.TotalCents)
	}
}

func TestConcurrentDuplicateOrderSubmissionsReplayOneOrder(t *testing.T) {
	s := newTestServer(t)
	token := s.registerAndLogin("owner-" + uuid.NewString() + "@example.test")
	_, _, qrToken, _, itemID := s.createTenant(token, "Idempotency Cafe")
	sessionToken := s.scan(qrToken)

	key := uuid.NewString()
	body := map[string]any{
		"idempotency_key": key,
		"items": []map[string]any{{
			"menu_item_id": itemID,
			"quantity":     1,
		}},
	}

	const workers = 12
	var wg sync.WaitGroup
	results := make(chan string, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var resp struct {
				ID string `json:"id"`
			}
			s.request(http.MethodPost, "/api/v1/public/sessions/"+sessionToken+"/orders", "", body, http.StatusCreated, &resp)
			results <- resp.ID
		}()
	}
	wg.Wait()
	close(results)

	ids := map[string]bool{}
	for id := range results {
		ids[id] = true
	}
	if len(ids) != 1 {
		t.Fatalf("expected one idempotent order id, got %#v", ids)
	}

	var count int64
	if err := s.db.Table("orders").Where("idempotency_key = ?", key).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one DB order for idempotency key, got %d", count)
	}
}

func TestConcurrentScanCreatesSingleActiveSession(t *testing.T) {
	s := newTestServer(t)
	token := s.registerAndLogin("owner-" + uuid.NewString() + "@example.test")
	_, tableID, qrToken, _, _ := s.createTenant(token, "Concurrent Cafe")

	const workers = 20
	var wg sync.WaitGroup
	results := make(chan string, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- s.scan(qrToken)
		}()
	}
	wg.Wait()
	close(results)

	seen := map[string]bool{}
	for token := range results {
		seen[token] = true
	}
	if len(seen) != 1 {
		t.Fatalf("expected one reused active session, got %d: %#v", len(seen), seen)
	}

	var count int64
	deadline := time.Now().Add(2 * time.Second)
	for {
		if err := s.db.Table("sessions").Where("table_id = ? AND status = 'ACTIVE'", tableID).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count == 1 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if count != 1 {
		t.Fatalf("expected one active session row, got %d", count)
	}
}

func TestWebSocketSessionAuthorization(t *testing.T) {
	s := newTestServer(t)
	token := s.registerAndLogin("owner-" + uuid.NewString() + "@example.test")
	restaurantID, _, qrToken, _, _ := s.createTenant(token, "WebSocket Cafe")
	sessionToken := s.scan(qrToken)
	server := httptest.NewServer(s.router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/sessions/" + sessionToken
	header := http.Header{"Origin": []string{"https://app.example.test"}}
	conn, _, err := gorilla.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("expected valid websocket session to connect: %v", err)
	}
	_ = conn.Close()

	badOrigin := http.Header{"Origin": []string{"https://evil.example.test"}}
	if conn, _, err := gorilla.DefaultDialer.Dial(wsURL, badOrigin); err == nil {
		_ = conn.Close()
		t.Fatal("expected foreign origin websocket connection to be rejected")
	}

	missingURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/sessions/" + uuid.NewString()
	if conn, _, err := gorilla.DefaultDialer.Dial(missingURL, header); err == nil {
		_ = conn.Close()
		t.Fatal("expected nonexistent session websocket connection to be rejected")
	}

	s.request(http.MethodPost, fmt.Sprintf("/api/v1/restaurants/%s/sessions/%s/close", restaurantID, sessionTokenToID(t, s.db, sessionToken)), token, map[string]any{
		"force": true,
	}, http.StatusOK, &map[string]any{})
	if conn, _, err := gorilla.DefaultDialer.Dial(wsURL, header); err == nil {
		_ = conn.Close()
		t.Fatal("expected closed session websocket connection to be rejected")
	}
}

func TestExpiredSessionCannotOrderOrConnectAndQRIsReusable(t *testing.T) {
	s := newTestServer(t)
	token := s.registerAndLogin("owner-" + uuid.NewString() + "@example.test")
	_, _, qrToken, _, itemID := s.createTenant(token, "Expiry Cafe")
	sessionToken := s.scan(qrToken)

	if err := s.db.Table("sessions").
		Where("session_token = ?", sessionToken).
		Updates(map[string]any{
			"expires_at":   time.Now().Add(-time.Minute),
			"last_seen_at": time.Now().Add(-time.Hour),
		}).Error; err != nil {
		t.Fatal(err)
	}

	s.request(http.MethodPost, "/api/v1/public/sessions/"+sessionToken+"/orders", "", map[string]any{
		"items": []map[string]any{{
			"menu_item_id": itemID,
			"quantity":     1,
		}},
	}, http.StatusForbidden, nil)
	s.request(http.MethodGet, "/api/v1/public/sessions/"+sessionToken+"/menu", "", nil, http.StatusForbidden, nil)

	server := httptest.NewServer(s.router)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/sessions/" + sessionToken
	header := http.Header{"Origin": []string{"https://app.example.test"}}
	if conn, _, err := gorilla.DefaultDialer.Dial(wsURL, header); err == nil {
		_ = conn.Close()
		t.Fatal("expected expired session websocket connection to be rejected")
	}

	newSessionToken := s.scan(qrToken)
	if newSessionToken == sessionToken {
		t.Fatal("expected QR scan to replace expired active session")
	}
	var activeCount int64
	if err := s.db.Table("sessions").Where("status = 'ACTIVE'").Count(&activeCount).Error; err != nil {
		t.Fatal(err)
	}
	if activeCount != 1 {
		t.Fatalf("expected one active session after expired replacement, got %d", activeCount)
	}
}

func TestAuditLogCreatedForAdminMutations(t *testing.T) {
	s := newTestServer(t)
	token := s.registerAndLogin("owner-" + uuid.NewString() + "@example.test")
	restaurantID, tableID, qrToken, _, itemID := s.createTenant(token, "Audit Cafe")
	sessionToken := s.scan(qrToken)

	var orderResp struct {
		ID string `json:"id"`
	}
	s.request(http.MethodPost, "/api/v1/public/sessions/"+sessionToken+"/orders", "", map[string]any{
		"items": []map[string]any{{
			"menu_item_id": itemID,
			"quantity":     1,
		}},
	}, http.StatusCreated, &orderResp)

	s.request(http.MethodPatch, fmt.Sprintf("/api/v1/restaurants/%s/orders/%s/status", restaurantID, orderResp.ID), token, map[string]any{
		"status": "ACCEPTED",
	}, http.StatusOK, &map[string]any{})
	s.request(http.MethodPost, fmt.Sprintf("/api/v1/restaurants/%s/tables/%s/regenerate-qr", restaurantID, tableID), token, nil, http.StatusOK, &map[string]any{})
	s.request(http.MethodPatch, fmt.Sprintf("/api/v1/restaurants/%s/menu/items/%s", restaurantID, itemID), token, map[string]any{
		"options_config": map[string]any{"modifier_groups": []map[string]any{}},
	}, http.StatusOK, &map[string]any{})

	for _, action := range []string{"order.status_update", "table.qr_regenerate", "menu_item.modifiers_update"} {
		var count int64
		if err := s.db.Table("audit_logs").Where("tenant_id = ? AND action = ?", restaurantID, action).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("expected one audit log for %s, got %d", action, count)
		}
	}
}

func sessionTokenToID(t *testing.T, db *gorm.DB, token string) string {
	t.Helper()
	var id string
	if err := db.Table("sessions").Select("id").Where("session_token = ?", token).Scan(&id).Error; err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatalf("session not found for token %s", token)
	}
	return id
}

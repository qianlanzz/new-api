package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"gorm.io/gorm"
)

type provisionUserTokenResponse struct {
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Quota     int    `json:"quota"`
	TokenID   int    `json:"token_id"`
	TokenName string `json:"token_name"`
	Key       string `json:"key"`
}

type queryUserKeysResponse struct {
	UserID   int                          `json:"user_id"`
	Username string                       `json:"username"`
	Key      string                       `json:"key"`
	Tokens   []service.ProvisionTokenItem `json:"tokens"`
}

func setupUserAdminTokenTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := openTokenControllerTestDB(t)
	if err := db.AutoMigrate(&model.User{}, &model.Token{}); err != nil {
		t.Fatalf("failed to migrate user/token tables: %v", err)
	}
	return db
}

func TestProvisionUserTokenCreatesUserAndReturnsFullKey(t *testing.T) {
	db := setupUserAdminTokenTestDB(t)

	originalQuotaForNewUser := common.QuotaForNewUser
	originalAutoGroup := setting.DefaultUseAutoGroup
	common.QuotaForNewUser = 0
	setting.DefaultUseAutoGroup = false
	t.Cleanup(func() {
		common.QuotaForNewUser = originalQuotaForNewUser
		setting.DefaultUseAutoGroup = originalAutoGroup
	})

	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/user/provision", map[string]any{
		"username": "alice",
	}, 1)
	ProvisionUserToken(ctx)

	response := decodeAPIResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	var data provisionUserTokenResponse
	if err := common.Unmarshal(response.Data, &data); err != nil {
		t.Fatalf("failed to decode provision response: %v", err)
	}
	if data.Username != "alice" {
		t.Fatalf("expected username alice, got %q", data.Username)
	}
	if data.Password != service.DefaultProvisionPassword {
		t.Fatalf("expected password %q, got %q", service.DefaultProvisionPassword, data.Password)
	}
	if data.Quota != service.DefaultProvisionQuota {
		t.Fatalf("expected quota %d, got %d", service.DefaultProvisionQuota, data.Quota)
	}
	if data.TokenName != service.DefaultProvisionTokenName {
		t.Fatalf("expected token name %q, got %q", service.DefaultProvisionTokenName, data.TokenName)
	}
	if data.Key == "" {
		t.Fatalf("expected non-empty full key in response")
	}

	user, err := model.GetUserByUsername("alice", true)
	if err != nil {
		t.Fatalf("failed to load created user: %v", err)
	}
	if user.Quota != service.DefaultProvisionQuota {
		t.Fatalf("expected user quota %d, got %d", service.DefaultProvisionQuota, user.Quota)
	}
	if !common.ValidatePasswordAndHash(service.DefaultProvisionPassword, user.Password) {
		t.Fatalf("stored password hash does not match default password")
	}

	tokens, err := model.GetUserTokensByUserId(user.Id)
	if err != nil {
		t.Fatalf("failed to load created tokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected exactly one token, got %d", len(tokens))
	}
	if tokens[0].Name != service.DefaultProvisionTokenName {
		t.Fatalf("expected stored token name %q, got %q", service.DefaultProvisionTokenName, tokens[0].Name)
	}
	if tokens[0].RemainQuota != service.DefaultProvisionQuota {
		t.Fatalf("expected stored token quota %d, got %d", service.DefaultProvisionQuota, tokens[0].RemainQuota)
	}
	if tokens[0].GetFullKey() != data.Key {
		t.Fatalf("expected response key %q, got %q", tokens[0].GetFullKey(), data.Key)
	}

	if err := db.First(&model.Token{}, "id = ?", data.TokenID).Error; err != nil {
		t.Fatalf("expected created token id %d to exist: %v", data.TokenID, err)
	}
}

func TestQueryUserKeysReturnsFullKeysByUsername(t *testing.T) {
	setupUserAdminTokenTestDB(t)

	originalQuotaForNewUser := common.QuotaForNewUser
	originalAutoGroup := setting.DefaultUseAutoGroup
	common.QuotaForNewUser = 0
	setting.DefaultUseAutoGroup = false
	t.Cleanup(func() {
		common.QuotaForNewUser = originalQuotaForNewUser
		setting.DefaultUseAutoGroup = originalAutoGroup
	})

	created, err := service.ProvisionUserToken("bob")
	if err != nil {
		t.Fatalf("failed to provision seed user: %v", err)
	}

	user, err := model.GetUserByUsername("bob", false)
	if err != nil {
		t.Fatalf("failed to load seed user: %v", err)
	}

	extraToken := &model.Token{
		UserId:             user.Id,
		Name:               "extra",
		Key:                "extra-key-1234567890",
		Status:             common.TokenStatusEnabled,
		CreatedTime:        common.GetTimestamp() + 1,
		AccessedTime:       common.GetTimestamp() + 1,
		ExpiredTime:        -1,
		RemainQuota:        4321,
		UnlimitedQuota:     false,
		ModelLimitsEnabled: false,
		Group:              "default",
	}
	if err := model.DB.Create(extraToken).Error; err != nil {
		t.Fatalf("failed to create extra token: %v", err)
	}

	ctx, recorder := newAuthenticatedContext(t, http.MethodPost, "/api/user/query_keys", map[string]any{
		"username": "bob",
	}, 1)
	QueryUserKeys(ctx)

	response := decodeAPIResponse(t, recorder)
	if !response.Success {
		t.Fatalf("expected success response, got message: %s", response.Message)
	}

	var data queryUserKeysResponse
	if err := common.Unmarshal(response.Data, &data); err != nil {
		t.Fatalf("failed to decode query response: %v", err)
	}
	if data.Username != "bob" {
		t.Fatalf("expected username bob, got %q", data.Username)
	}
	if data.UserID != created.UserID {
		t.Fatalf("expected user id %d, got %d", created.UserID, data.UserID)
	}
	if len(data.Tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(data.Tokens))
	}
	if data.Key != extraToken.Key {
		t.Fatalf("expected latest key %q, got %q", extraToken.Key, data.Key)
	}
	if data.Tokens[0].Key != extraToken.Key {
		t.Fatalf("expected first token key %q, got %q", extraToken.Key, data.Tokens[0].Key)
	}
	if data.Tokens[1].Key != created.Key {
		t.Fatalf("expected second token key %q, got %q", created.Key, data.Tokens[1].Key)
	}
}

package service

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"gorm.io/gorm"
)

const (
	DefaultProvisionPassword  = "123456"
	DefaultProvisionQuota     = 10000
	DefaultProvisionTokenName = "test"
)

type ProvisionTokenItem struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	RemainQuota int    `json:"remain_quota"`
	CreatedTime int64  `json:"created_time"`
}

type ProvisionUserTokenResult struct {
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Quota     int    `json:"quota"`
	TokenID   int    `json:"token_id"`
	TokenName string `json:"token_name"`
	Key       string `json:"key"`
}

type QueryUserTokensResult struct {
	UserID   int                  `json:"user_id"`
	Username string               `json:"username"`
	Key      string               `json:"key"`
	Tokens   []ProvisionTokenItem `json:"tokens"`
}

func normalizeProvisionUsername(username string) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return "", errors.New("用户名不能为空")
	}
	if utf8.RuneCountInString(username) > model.UserNameMaxLength {
		return "", fmt.Errorf("用户名长度不能超过 %d 个字符", model.UserNameMaxLength)
	}
	return username, nil
}

func ProvisionUserToken(username string) (*ProvisionUserTokenResult, error) {
	username, err := normalizeProvisionUsername(username)
	if err != nil {
		return nil, err
	}

	exists, err := model.CheckUserExistOrDeleted(username, "")
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("用户已存在")
	}

	tx := model.DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	user := &model.User{
		Username:    username,
		Password:    DefaultProvisionPassword,
		DisplayName: username,
		Role:        common.RoleCommonUser,
	}
	if err := user.InsertWithTx(tx, 0); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Model(&model.User{}).Where("id = ?", user.Id).Update("quota", DefaultProvisionQuota).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	key, err := common.GenerateKey()
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	now := common.GetTimestamp()
	token := &model.Token{
		UserId:             user.Id,
		Name:               DefaultProvisionTokenName,
		Key:                key,
		CreatedTime:        now,
		AccessedTime:       now,
		ExpiredTime:        -1,
		RemainQuota:        DefaultProvisionQuota,
		UnlimitedQuota:     false,
		ModelLimitsEnabled: false,
	}
	if setting.DefaultUseAutoGroup {
		token.Group = "auto"
	}
	if err := tx.Create(token).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	if err := model.InitializeUserSidebarConfig(user.Id); err != nil {
		common.SysLog(fmt.Sprintf("为管理员接口创建的用户 %s 初始化边栏配置失败: %s", user.Username, err.Error()))
	}

	return &ProvisionUserTokenResult{
		UserID:    user.Id,
		Username:  user.Username,
		Password:  DefaultProvisionPassword,
		Quota:     DefaultProvisionQuota,
		TokenID:   token.Id,
		TokenName: token.Name,
		Key:       token.GetFullKey(),
	}, nil
}

func QueryUserTokensByUsername(username string) (*QueryUserTokensResult, error) {
	username, err := normalizeProvisionUsername(username)
	if err != nil {
		return nil, err
	}

	user, err := model.GetUserByUsername(username, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}

	tokens, err := model.GetUserTokensByUserId(user.Id)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, errors.New("该用户暂无 key")
	}

	items := make([]ProvisionTokenItem, 0, len(tokens))
	for _, token := range tokens {
		items = append(items, ProvisionTokenItem{
			ID:          token.Id,
			Name:        token.Name,
			Key:         token.GetFullKey(),
			RemainQuota: token.RemainQuota,
			CreatedTime: token.CreatedTime,
		})
	}

	return &QueryUserTokensResult{
		UserID:   user.Id,
		Username: user.Username,
		Key:      items[0].Key,
		Tokens:   items,
	}, nil
}

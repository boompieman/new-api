package model

import (
	"context"
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// RequestContent lives in the primary database, including when LOG_DB is ClickHouse.
// Bodies are bounded JSON text, never headers or binary attachments.
type RequestContent struct {
	ID         int    `json:"-"`
	RequestId  string `json:"request_id" gorm:"size:64;uniqueIndex"`
	UserId     int    `json:"-" gorm:"index"`
	TokenId    int    `json:"token_id"`
	Model      string `json:"model" gorm:"size:255"`
	ExpiresAt  int64  `json:"expires_at" gorm:"index"`
	Preview    string `json:"preview" gorm:"type:text"`
	Input      string `json:"input" gorm:"type:text"`
	Output     string `json:"output" gorm:"type:text"`
	Truncated  bool   `json:"truncated"`
	Complete   bool   `json:"complete"`
	StatusCode int    `json:"status_code"`
}

func SaveRequestContent(ctx context.Context, entry *RequestContent) error {
	if entry.UserId <= 0 || entry.RequestId == "" {
		return errors.New("request content requires an owner and request ID")
	}
	return DB.Session(&gorm.Session{Logger: logger.Discard}).WithContext(ctx).Create(entry).Error
}

func GetRequestContent(ctx context.Context, requestId string, userId int, admin bool) (*RequestContent, error) {
	if !admin && userId <= 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if !admin {
		allowed, err := CanViewRequestContent(ctx, userId)
		if err != nil {
			return nil, err
		}
		if !allowed {
			return nil, gorm.ErrRecordNotFound
		}
	}
	tx := DB.WithContext(ctx).Where("request_id = ? AND expires_at > ?", requestId, common.GetTimestamp())
	if !admin {
		tx = tx.Where("user_id = ?", userId)
	}
	var entry RequestContent
	if err := tx.First(&entry).Error; err != nil {
		return nil, err
	}
	return &entry, nil
}

// AddRequestContentPreviews never loads message bodies into the paginated log list.
func AddRequestContentPreviews(ctx context.Context, logs []*Log) error {
	ids := make([]string, 0, len(logs))
	for _, log := range logs {
		if log.RequestId != "" {
			ids = append(ids, log.RequestId)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	var entries []RequestContent
	if err := DB.WithContext(ctx).Select("request_id", "user_id", "preview").Where("request_id IN ? AND expires_at > ?", ids, common.GetTimestamp()).Find(&entries).Error; err != nil {
		return err
	}
	byID := make(map[string]RequestContent, len(entries))
	for _, entry := range entries {
		byID[entry.RequestId] = entry
	}
	for _, log := range logs {
		if entry, ok := byID[log.RequestId]; ok && entry.UserId == log.UserId {
			log.RequestPreview = entry.Preview
			log.HasRequestContent = true
		}
	}
	return nil
}

func DeleteExpiredRequestContent(ctx context.Context) error {
	// Drain expired rows in bounded batches until the caller's deadline.
	for {
		var ids []int
		if err := DB.WithContext(ctx).Model(&RequestContent{}).Where("expires_at <= ?", common.GetTimestamp()).Limit(500).Pluck("id", &ids).Error; err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		if err := DB.WithContext(ctx).Where("id IN ?", ids).Delete(&RequestContent{}).Error; err != nil {
			return err
		}
		if len(ids) < 500 {
			return nil
		}
	}
}

func PruneRequestContent() {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := DeleteExpiredRequestContent(ctx); err != nil {
			common.SysError("request content retention failed: " + err.Error())
		}
		cancel()
		time.Sleep(time.Minute)
	}
}

// A row grants customer access. Absence denies it; recording is independent.
type RequestContentAccess struct {
	UserId int `gorm:"primaryKey;autoIncrement:false"`
}

func CanViewRequestContent(ctx context.Context, userId int) (bool, error) {
	if userId <= 0 {
		return false, nil
	}
	var count int64
	err := DB.WithContext(ctx).Model(&RequestContentAccess{}).Where("user_id = ?", userId).Count(&count).Error
	return count > 0, err
}

func SetRequestContentAccess(ctx context.Context, userId int, enabled bool) error {
	if userId <= 0 {
		return errors.New("invalid user ID")
	}
	if !enabled {
		return DB.WithContext(ctx).Where("user_id = ?", userId).Delete(&RequestContentAccess{}).Error
	}
	return DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&RequestContentAccess{UserId: userId}).Error
}

package orderapp

import (
	"database/sql"
	"encoding/json"
	"example_shop/common/db"
	model2 "example_shop/internal/model"
	kitexuser "example_shop/kitex_gen/user"
	"errors"
	"strings"
	"time"

	mysqlerr "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func baseResp(code int32, msg string) *kitexuser.BaseResp {
	return &kitexuser.BaseResp{Code: code, Msg: msg}
}

func nullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: true}
}

func nullString(s string) sql.NullString {
	if strings.TrimSpace(s) == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	var me *mysqlerr.MySQLError
	if errors.As(err, &me) && me != nil && me.Number == 1062 {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate entry") || strings.Contains(msg, "duplicate")
}

// writeOrderAuditLog 写入订单审计日志。
//
// 审计日志属于“辅助可观测”能力：写入失败不应影响主流程，所以这里采用 best-effort。
func writeOrderAuditLog(tx *gorm.DB, orderID, operateType, userID, before, after string, detail any) {
	var beforeNS, afterNS sql.NullString
	if strings.TrimSpace(before) != "" {
		beforeNS = sql.NullString{String: before, Valid: true}
	}
	if strings.TrimSpace(after) != "" {
		afterNS = sql.NullString{String: after, Valid: true}
	}
	var detailJSON *model2.JSON
	if detail != nil {
		if b, err := json.Marshal(detail); err == nil {
			if j, err2 := model2.ToJSON(json.RawMessage(b)); err2 == nil {
				detailJSON = &j
			}
		}
	}
	logRow := model2.OrderAuditLog{
		OrderID:       orderID,
		OperateType:   operateType,
		OperateUser:   userID,
		BeforeStatus:  beforeNS,
		AfterStatus:   afterNS,
		OperateDetail: detailJSON,
		TraceID:       uuid.New().String(),
		CreatedAt:     time.Now(),
	}
	_ = tx.Create(&logRow).Error
}

func readDB() *gorm.DB {
	return db.ReadDB()
}

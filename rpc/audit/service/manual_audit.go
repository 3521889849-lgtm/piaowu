package service

import (
	"context"
	"encoding/json"
	"errors"
	"piaowu/common/db"
	"piaowu/common/model/audit"
	audit_kitex "piaowu/kitex_gen/audit"
	"piaowu/rpc/audit/component/metrics"
	"strconv"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	"gorm.io/gorm"
)

type ManualAuditService struct{}

var ManualAuditSvc = new(ManualAuditService)

// FetchManualTasks 获取人工审核任务列表
func (s *ManualAuditService) FetchManualTasks(ctx context.Context, req *audit_kitex.FetchManualTasksReq) (*audit_kitex.FetchManualTasksResp, error) {
	resp := new(audit_kitex.FetchManualTasksResp)
	resp.BaseResp = new(audit_kitex.BaseResp)

	var auditMains []audit.AuditMain
	var total int64

	// 构建查询条件
	query := db.DB.Model(&audit.AuditMain{})

	// 1. 筛选状态 (PendingManual 通常对应 PENDING)
	if req.Status != 0 {
		query = query.Where("audit_status = ?", req.Status)
	} else {
		// 默认只查待审核
		query = query.Where("audit_status = ?", audit_kitex.AuditStatus_PENDING)
	}

	// 2. 筛选业务类型
	if len(req.BizTypes) > 0 {
		bizTypes := make([]int8, 0, len(req.BizTypes))
		for _, bt := range req.BizTypes {
			bizTypes = append(bizTypes, int8(bt))
		}
		query = query.Where("business_type IN ?", bizTypes)
	}

	// 3. 筛选审核人 (可选)
	if req.AuditorId != "" {
		// 如果有指定审核人，查询分配给该人的（假设 AuditUserId 字段用于分配）
		// 注意：这里的逻辑取决于是否已经预分配。如果未分配，可能不需要这个条件。
		// 暂时实现为：查询已分配给该用户的 或者 未分配的
		query = query.Where("audit_user_id = ? OR audit_user_id = 0", req.AuditorId)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		klog.Errorf("Count tasks failed: %v", err)
		resp.BaseResp.Code = 500
		resp.BaseResp.Msg = "查询失败"
		return resp, nil
	}
	resp.Total = total

	// 分页查询
	pageNum := 1
	pageSize := 10
	if req.Pagination != nil {
		if req.Pagination.PageNum > 0 {
			pageNum = int(req.Pagination.PageNum)
		}
		if req.Pagination.PageSize > 0 {
			pageSize = int(req.Pagination.PageSize)
		}
	}
	offset := (pageNum - 1) * pageSize

	if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&auditMains).Error; err != nil {
		klog.Errorf("Query tasks failed: %v", err)
		resp.BaseResp.Code = 500
		resp.BaseResp.Msg = "查询失败"
		return resp, nil
	}

	// 转换为 IDL 结构
	tasks := make([]*audit_kitex.AuditTask, 0, len(auditMains))
	for _, m := range auditMains {
		task := &audit_kitex.AuditTask{
			AuditId:     int64(m.ID),
			BizType:     audit_kitex.BizType(m.BusinessType),
			BizId:       strconv.FormatUint(m.BusinessId, 10),
			SubmitterId: strconv.FormatUint(m.SubmitUserId, 10),
			Content:     m.AuditRemark, // 暂时使用备注作为内容简述，因为 Content 不在主表
			ApplyTime:   m.CreatedAt.Format(time.RFC3339),
			Status:      audit_kitex.AuditStatus(m.AuditStatus),
			Priority:    1, // 默认为1
		}
		tasks = append(tasks, task)
	}
	resp.Tasks = tasks
	resp.BaseResp.Code = 0
	resp.BaseResp.Msg = "Success"

	return resp, nil
}

// ProcessManualAudit 处理人工审核结果
func (s *ManualAuditService) ProcessManualAudit(ctx context.Context, req *audit_kitex.ProcessManualAuditReq) (*audit_kitex.ProcessManualAuditResp, error) {
	resp := new(audit_kitex.ProcessManualAuditResp)
	resp.BaseResp = new(audit_kitex.BaseResp)

	// 校验参数
	if req.AuditId == 0 {
		resp.BaseResp.Code = 400
		resp.BaseResp.Msg = "AuditId is required"
		return resp, nil
	}

	var status audit_kitex.AuditStatus
	if req.IsPassed {
		status = audit_kitex.AuditStatus_PASSED
	} else {
		status = audit_kitex.AuditStatus_REJECTED
	}

	var suggestedAction metrics.DecisionType

	err := db.DB.Transaction(func(tx *gorm.DB) error {

		// 1. 检查工单是否存在及状态
		var auditMain audit.AuditMain
		if err := tx.First(&auditMain, req.AuditId).Error; err != nil {
			return err
		}

		// 读取自动审核建议（用于准确率统计）
		suggestedAction = parseSuggestedAction(auditMain.Extra)

		if audit_kitex.AuditStatus(auditMain.AuditStatus) != audit_kitex.AuditStatus_PENDING &&

			audit_kitex.AuditStatus(auditMain.AuditStatus) != audit_kitex.AuditStatus_PROCESSING {
			return gorm.ErrInvalidData // 状态不正确
		}

		// 2. 更新主表状态
		now := time.Now()
		updates := map[string]interface{}{
			"audit_status":    int8(status),
			"audit_user_id":   req.AuditorId, // 假设 AuditorId 是数字字符串，需转换，这里暂存为0如果转换失败
			"audit_user_name": "Operator_" + req.AuditorId,
			"audit_remark":    req.Remark,
			"audit_time":      now,
			"updated_at":      now,
		}
		if uid, err := strconv.ParseUint(req.AuditorId, 10, 64); err == nil {
			updates["audit_user_id"] = uid
		}

		if err := tx.Model(&auditMain).Updates(updates).Error; err != nil {
			return err
		}

		// 3. 插入操作日志
		opLog := &audit.AuditOperationLog{
			AuditMainId:     auditMain.ID,
			OperatorId:      uint64(updates["audit_user_id"].(uint64)), // 简化处理
			OperatorName:    updates["audit_user_name"].(string),
			OperationType:   3, // 3=通过, 4=驳回
			OperationRemark: req.Remark,
			CreatedAt:       now,
		}
		if !req.IsPassed {
			opLog.OperationType = 4
		}

		if err := tx.Create(opLog).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		klog.Errorf("Process manual audit failed: %v", err)
		resp.BaseResp.Code = 500
		resp.BaseResp.Msg = "处理失败: " + err.Error()
		return resp, nil
	}

	// 记录人工审核准确率（对比自动建议与最终结果）
	metrics.DefaultCollector.RecordManualOutcome(suggestedAction, statusToDecision(status))

	klog.Infof("Manual audit processed. ID: %d, Passed: %v, Auditor: %s", req.AuditId, req.IsPassed, req.AuditorId)

	resp.BaseResp.Code = 0
	resp.BaseResp.Msg = "Success"
	return resp, nil
}

// GetAuditRecord 查询审核记录详情
func (s *ManualAuditService) GetAuditRecord(ctx context.Context, req *audit_kitex.GetAuditRecordReq) (*audit_kitex.GetAuditRecordResp, error) {
	resp := new(audit_kitex.GetAuditRecordResp)
	resp.BaseResp = new(audit_kitex.BaseResp)

	var auditMain audit.AuditMain
	query := db.DB.Model(&audit.AuditMain{})

	if req.AuditId > 0 {
		query = query.Where("id = ?", req.AuditId)
	} else if req.BizId != "" && req.BizType != 0 {
		query = query.Where("business_id = ? AND business_type = ?", req.BizId, req.BizType)
	} else {
		resp.BaseResp.Code = 400
		resp.BaseResp.Msg = "Missing query params"
		return resp, nil
	}

	if err := query.First(&auditMain).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			resp.BaseResp.Code = 404
			resp.BaseResp.Msg = "Record not found"
			return resp, nil
		}
		klog.Errorf("Get audit record failed: %v", err)
		resp.BaseResp.Code = 500
		resp.BaseResp.Msg = "Query failed"
		return resp, nil
	}

	// 查询日志
	var logs []audit.AuditOperationLog
	if err := db.DB.Where("audit_main_id = ?", auditMain.ID).Order("created_at asc").Find(&logs).Error; err != nil {
		klog.Errorf("Get audit logs failed: %v", err)
	}

	// 组装 Detail
	detail := &audit_kitex.AuditDetail{
		TaskInfo: &audit_kitex.AuditTask{
			AuditId:     int64(auditMain.ID),
			BizType:     audit_kitex.BizType(auditMain.BusinessType),
			BizId:       strconv.FormatUint(auditMain.BusinessId, 10),
			SubmitterId: strconv.FormatUint(auditMain.SubmitUserId, 10),
			Content:     auditMain.AuditRemark,
			ApplyTime:   auditMain.CreatedAt.Format(time.RFC3339),
			Status:      audit_kitex.AuditStatus(auditMain.AuditStatus),
		},
		AuditResult_: auditMain.AuditRemark,
	}

	// 转换日志
	for _, l := range logs {
		opType := "未知"
		switch l.OperationType {
		case 1:
			opType = "提交"
		case 3:
			opType = "通过"
		case 4:
			opType = "驳回"
		}
		detail.Logs = append(detail.Logs, &audit_kitex.AuditLog{
			OperatorId: strconv.FormatUint(l.OperatorId, 10),
			Operation:  opType,
			Remark:     l.OperationRemark,
			CreateTime: l.CreatedAt.Format(time.RFC3339),
		})
	}

	resp.Detail = detail
	resp.BaseResp.Code = 0
	resp.BaseResp.Msg = "Success"
	return resp, nil
}

// 解析自动审核建议动作
func parseSuggestedAction(extra string) metrics.DecisionType {
	if extra == "" {
		return metrics.DecisionReview
	}
	var data struct {
		FinalAction string `json:"final_action"`
	}
	if err := json.Unmarshal([]byte(extra), &data); err != nil {
		return metrics.DecisionReview
	}
	switch data.FinalAction {
	case string(metrics.DecisionPass):
		return metrics.DecisionPass
	case string(metrics.DecisionReject):
		return metrics.DecisionReject
	case string(metrics.DecisionReview):
		return metrics.DecisionReview
	default:
		return metrics.DecisionReview
	}
}

func statusToDecision(status audit_kitex.AuditStatus) metrics.DecisionType {
	switch status {
	case audit_kitex.AuditStatus_PASSED:
		return metrics.DecisionPass
	case audit_kitex.AuditStatus_REJECTED:
		return metrics.DecisionReject
	default:
		return metrics.DecisionReview
	}
}

package main

import (
	"context"
	audit "piaowu/kitex_gen/audit"
	"piaowu/rpc/audit/service"
)

// AuditServiceImpl 实现了 IDL 中定义的最后一个服务接口。
type AuditServiceImpl struct{}

// ApplyAudit 实现了 AuditServiceImpl 接口。
func (s *AuditServiceImpl) ApplyAudit(ctx context.Context, req *audit.ApplyAuditReq) (resp *audit.ApplyAuditResp, err error) {
	resp = new(audit.ApplyAuditResp)
	resp.BaseResp = new(audit.BaseResp)

	// 调用自动审核服务
	auditID, err := service.AutoAuditSvc.AutoAudit(ctx, req)
	if err != nil {
		resp.BaseResp.Code = 500
		resp.BaseResp.Msg = "审核提交失败: " + err.Error()
		return resp, nil
	}

	resp.BaseResp.Code = 0
	resp.BaseResp.Msg = "提交成功"
	resp.AuditId = auditID
	return resp, nil
}

// FetchManualTasks 实现了 AuditServiceImpl 接口。
func (s *AuditServiceImpl) FetchManualTasks(ctx context.Context, req *audit.FetchManualTasksReq) (resp *audit.FetchManualTasksResp, err error) {
	return service.ManualAuditSvc.FetchManualTasks(ctx, req)
}

// ProcessManualAudit 实现了 AuditServiceImpl 接口。
func (s *AuditServiceImpl) ProcessManualAudit(ctx context.Context, req *audit.ProcessManualAuditReq) (resp *audit.ProcessManualAuditResp, err error) {
	return service.ManualAuditSvc.ProcessManualAudit(ctx, req)
}

// GetAuditRecord 实现了 AuditServiceImpl 接口。
func (s *AuditServiceImpl) GetAuditRecord(ctx context.Context, req *audit.GetAuditRecordReq) (resp *audit.GetAuditRecordResp, err error) {
	return service.ManualAuditSvc.GetAuditRecord(ctx, req)
}

package model

import (
	"time"
)

const TableNameSysMerchant = "sys_merchant"

// SysMerchant 商家信息表-对应商家入驻审核流程
type SysMerchant struct {
	BaseModel

	MerchantName     string     `gorm:"column:merchant_name;type:VARCHAR(100);not null;index:idx_merchant_name;comment:商家名称(景区/文旅公司)" json:"merchant_name"`
	EnterpriseCode   string     `gorm:"column:enterprise_code;type:VARCHAR(50);not null;uniqueIndex:uk_enterprise_code;comment:企业统一信用代码" json:"enterprise_code"`
	LegalPerson      string     `gorm:"column:legal_person;type:VARCHAR(30);not null;comment:法人姓名" json:"legal_person"`
	Phone            string     `gorm:"column:phone;type:VARCHAR(20);not null;comment:联系电话" json:"phone"`
	Address          string     `gorm:"column:address;type:VARCHAR(255);not null;comment:商家地址" json:"address"`
	QualificationImg string     `gorm:"column:qualification_img;type:VARCHAR(512);not null;comment:资质证明图片地址" json:"qualification_img"`
	AuditStatus      string     `gorm:"column:audit_status;type:VARCHAR(20);not null;default:INITIAL;index:idx_audit_status;comment:审核状态：INITIAL-待审核，APPROVED-通过，REJECTED-驳回" json:"audit_status"`
	RejectReason     *string    `gorm:"column:reject_reason;type:VARCHAR(512);comment:驳回理由，审核驳回时必填" json:"reject_reason,omitempty"`
	AdminID          *uint64    `gorm:"column:admin_id;type:BIGINT UNSIGNED;index:idx_admin_id;comment:审核管理员ID" json:"admin_id,omitempty"`
	AuditTime        *time.Time `gorm:"column:audit_time;type:DATETIME;comment:审核时间" json:"audit_time,omitempty"`
	ExtFields        *JSON      `gorm:"column:ext_fields;type:JSON;comment:扩展字段，预留未来业务扩展" json:"ext_fields,omitempty"`

	Admin *SysAdmin  `gorm:"foreignKey:AdminID;references:ID" json:"admin,omitempty"`
	Spots []SpotInfo `gorm:"foreignKey:MerchantID;references:ID" json:"spots,omitempty"`
}

func (SysMerchant) TableName() string { return TableNameSysMerchant }

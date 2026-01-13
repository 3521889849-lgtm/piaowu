package model

const TableNameSysAdmin = "sys_admin"

// SysAdmin 管理员信息表-管理端所有操作的执行人
type SysAdmin struct {
	BaseModel

	AdminName string `gorm:"column:admin_name;type:VARCHAR(50);not null;comment:管理员姓名" json:"admin_name"`
	Account   string `gorm:"column:account;type:VARCHAR(50);not null;uniqueIndex:uk_account;comment:登录账号" json:"account"`
	Password  string `gorm:"column:password;type:VARCHAR(64);not null;comment:登录密码（加密存储）" json:"-"`
	Phone     string `gorm:"column:phone;type:VARCHAR(20);not null;comment:联系电话" json:"phone"`
	Role      string `gorm:"column:role;type:VARCHAR(20);not null;comment:角色：SUPER-超级管理员，OPERATOR-运营管理员" json:"role"`
	Status    uint8  `gorm:"column:status;type:TINYINT UNSIGNED;not null;default:1;comment:状态：0-禁用，1-启用" json:"status"`

	Merchants []SysMerchant `gorm:"foreignKey:AdminID;references:ID" json:"merchants,omitempty"`
	OperLogs  []SysOperLog  `gorm:"foreignKey:OperAdminID;references:ID" json:"oper_logs,omitempty"`
}

func (SysAdmin) TableName() string { return TableNameSysAdmin }

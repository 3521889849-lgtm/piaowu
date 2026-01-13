package model

const TableNameSysUser = "sys_user"

// SysUser 用户信息表-实名制信息存储
type SysUser struct {
	BaseModel

	UserName  string  `gorm:"column:user_name;type:VARCHAR(50);not null;comment:用户名" json:"user_name"`
	Phone     string  `gorm:"column:phone;type:VARCHAR(20);not null;uniqueIndex:uk_phone;comment:手机号（登录账号）" json:"phone"`
	IDCard    *string `gorm:"column:id_card;type:VARCHAR(20);index:idx_id_card;comment:身份证号，实名制必填" json:"id_card,omitempty"`
	RealName  *string `gorm:"column:real_name;type:VARCHAR(30);comment:真实姓名，实名制必填" json:"real_name,omitempty"`
	Avatar    *string `gorm:"column:avatar;type:VARCHAR(512);comment:用户头像" json:"avatar,omitempty"`
	ExtFields *JSON   `gorm:"column:ext_fields;type:JSON;comment:扩展字段，如会员等级、积分等" json:"ext_fields,omitempty"`

	Travelers []Traveler `gorm:"foreignKey:UserID;references:ID" json:"travelers,omitempty"`
}

func (SysUser) TableName() string { return TableNameSysUser }

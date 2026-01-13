package model

const TableNameTraveler = "traveler"

// Traveler 出行人信息表-用户下单时必填
type Traveler struct {
	BaseModel

	UserID    uint64 `gorm:"column:user_id;type:BIGINT UNSIGNED;not null;index:idx_user_id;comment:所属用户ID，关联sys_user.id" json:"user_id"`
	RealName  string `gorm:"column:real_name;type:VARCHAR(30);not null;comment:出行人真实姓名" json:"real_name"`
	IDCard    string `gorm:"column:id_card;type:VARCHAR(20);not null;index:idx_traveler_id_card;comment:出行人身份证号" json:"id_card"`
	Phone     string `gorm:"column:phone;type:VARCHAR(20);not null;comment:出行人手机号" json:"phone"`
	IsDefault uint8  `gorm:"column:is_default;type:TINYINT UNSIGNED;not null;default:0;comment:是否默认出行人：0-否，1-是" json:"is_default"`

	User *SysUser `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

func (Traveler) TableName() string { return TableNameTraveler }

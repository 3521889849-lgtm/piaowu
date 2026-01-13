package model

const TableNameSpotInfo = "spot_info"

// SpotInfo 景点信息表-门票查询的基础数据
type SpotInfo struct {
	BaseModel

	MerchantID   uint64  `gorm:"column:merchant_id;type:BIGINT UNSIGNED;not null;index:idx_merchant_id;comment:所属商家ID，关联sys_merchant.id" json:"merchant_id"`
	SpotName     string  `gorm:"column:spot_name;type:VARCHAR(100);not null;index:idx_spot_name;comment:景点名称" json:"spot_name"`
	SpotDesc     *string `gorm:"column:spot_desc;type:TEXT;comment:景点介绍" json:"spot_desc,omitempty"`
	Province     string  `gorm:"column:province;type:VARCHAR(30);not null;index:idx_province_city,priority:1;comment:省份" json:"province"`
	City         string  `gorm:"column:city;type:VARCHAR(30);not null;index:idx_province_city,priority:2;comment:城市" json:"city"`
	Address      string  `gorm:"column:address;type:VARCHAR(255);not null;comment:景点详细地址" json:"address"`
	CoverImg     string  `gorm:"column:cover_img;type:VARCHAR(512);not null;comment:景点封面图" json:"cover_img"`
	OpenTime     string  `gorm:"column:open_time;type:VARCHAR(100);not null;comment:开放时间" json:"open_time"`
	ContactPhone *string `gorm:"column:contact_phone;type:VARCHAR(20);comment:景点联系电话" json:"contact_phone,omitempty"`
	ExtFields    *JSON   `gorm:"column:ext_fields;type:JSON;comment:扩展字段，如评分、特色标签等" json:"ext_fields,omitempty"`

	Merchant   *SysMerchant `gorm:"foreignKey:MerchantID;references:ID" json:"merchant,omitempty"`
	TicketType []TicketType `gorm:"foreignKey:SpotID;references:ID" json:"ticket_types,omitempty"`
}

func (SpotInfo) TableName() string { return TableNameSpotInfo }

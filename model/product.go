package model

import (
	"time"

	"github.com/shemic/dever/orm"
)

type Product struct {
	ID                         uint64    `dorm:"primaryKey;autoIncrement;comment:产品ID"`
	Code                       string    `dorm:"type:varchar(64);not null;comment:产品编码"`
	Name                       string    `dorm:"type:varchar(128);not null;comment:产品名称"`
	Category                   string    `dorm:"type:varchar(32);not null;default:'consulting';comment:产品分类"`
	DefaultSigningBusinessType string    `dorm:"type:varchar(64);not null;default:'manual_review';comment:默认签约方向"`
	Description                string    `dorm:"type:text;not null;default:'';comment:说明"`
	NeedPMReview               bool      `dorm:"type:boolean;not null;default:true;comment:需要PM审核"`
	NeedLawyerReview           bool      `dorm:"type:boolean;not null;default:false;comment:需要律师审核"`
	NeedALAReview              bool      `dorm:"type:boolean;not null;default:false;comment:需要ALA审核"`
	NeedFinanceReview          bool      `dorm:"type:boolean;not null;default:false;comment:需要财务审核"`
	NeedContractReview         bool      `dorm:"type:boolean;not null;default:true;comment:需要合同审核"`
	Status                     int16     `dorm:"type:smallint;not null;default:1;comment:状态"`
	Sort                       int       `dorm:"type:int;not null;default:100;comment:排序"`
	CreatedAt                  time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdatedAt                  time.Time `dorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}

type ProductIndex struct {
	Code        struct{} `unique:"code"`
	StatusSort  struct{} `index:"status,sort,id"`
	Category    struct{} `index:"category,status,sort,id"`
	SigningType struct{} `index:"default_signing_business_type,status,id"`
}

func NewProductModel() *orm.Model[Product] {
	return orm.LoadModel[Product]("产品配置", "crm_product", orm.ModelConfig{
		Index:    ProductIndex{},
		Order:    "sort asc,id asc",
		Database: "default",
		Options: map[string]any{
			"category":                      productCategoryOptions,
			"default_signing_business_type": productSigningTypeOptions,
			"status":                        statusOptions,
		},
	})
}

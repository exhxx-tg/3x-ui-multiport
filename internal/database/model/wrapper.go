package model

type WrapperType string

const (
	WrapperWebSocket WrapperType = "websocket"
	WrapperTLS       WrapperType = "tls"
	WrapperHTTP2     WrapperType = "http2"
	WrapperGRPC      WrapperType = "grpc"
	WrapperNaive     WrapperType = "naive"
)

type TransportWrapper struct {
	Id                int         `json:"id" gorm:"primaryKey;autoIncrement"`
	WrapperType       WrapperType `json:"wrapperType" gorm:"column:wrapper_type;uniqueIndex:idx_wrapper_type_name;not null" validate:"required"`
	Name              string      `json:"name" gorm:"column:name;uniqueIndex:idx_wrapper_type_name;not null" validate:"required"`
	Remark            string      `json:"remark"`
	Protocols         string      `json:"protocols" gorm:"type:text"`
	Config            string      `json:"config" gorm:"type:text"`
	Enable            bool        `json:"enable" gorm:"default:true"`
	Port              int         `json:"port" gorm:"default:0" validate:"gte=0,lte=65535"`
	TlsEnabled        bool        `json:"tlsEnabled" gorm:"default:false"`
	CertFile          string      `json:"certFile"`
	KeyFile           string      `json:"keyFile"`
	CreatedAt         int64       `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt         int64       `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}

func (TransportWrapper) TableName() string { return "transport_wrappers" }

package model

type ServiceType string

const (
	ServiceTypeOpenVPN   ServiceType = "openvpn"
	ServiceTypeWireGuard ServiceType = "wireguard"
	ServiceTypeDropbear  ServiceType = "dropbear"
)

type ServiceStatus string

const (
	ServiceRunning    ServiceStatus = "running"
	ServiceStopped    ServiceStatus = "stopped"
	ServiceError      ServiceStatus = "error"
	ServiceInstalling ServiceStatus = "installing"
	ServiceUnknown    ServiceStatus = "unknown"
)

type Service struct {
	Id              int           `json:"id" gorm:"primaryKey;autoIncrement"`
	ServiceType     ServiceType   `json:"serviceType" gorm:"column:service_type;uniqueIndex:idx_service_type_name;not null" validate:"required"`
	Name            string        `json:"name" gorm:"column:name;uniqueIndex:idx_service_type_name;not null" validate:"required"`
	Remark          string        `json:"remark"`
	Port            int           `json:"port" gorm:"default:0" validate:"gte=0,lte=65535"`
	Protocol        string        `json:"protocol" gorm:"default:tcp"`
	Config          string        `json:"config" gorm:"type:text"`
	Status          ServiceStatus `json:"status" gorm:"default:unknown"`
	Enable          bool          `json:"enable" gorm:"default:true"`
	Up              int64         `json:"up" gorm:"default:0"`
	Down            int64         `json:"down" gorm:"default:0"`
	Total           int64         `json:"total" gorm:"default:0"`
	ExpiryTime      int64         `json:"expiryTime"`
	LastStartedAt   int64         `json:"lastStartedAt"`
	LastStoppedAt   int64         `json:"lastStoppedAt"`
	ErrorMsg        string        `json:"errorMsg" gorm:"type:text"`
	InstallPath     string        `json:"installPath"`
	ConfigPath      string        `json:"configPath"`
	CreatedAt       int64         `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt       int64         `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}

func (Service) TableName() string { return "services" }

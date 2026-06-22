package model

type Role string

const (
	RoleAdmin   Role = "admin"
	RoleOperator Role = "operator"
	RoleViewer  Role = "viewer"
	RoleService Role = "service"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleOperator, RoleViewer, RoleService:
		return true
	}
	return false
}

type Permission struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Resource  string `json:"resource" gorm:"column:resource;size:64;not null;uniqueIndex:idx_permission"`
	Action    string `json:"action" gorm:"column:action;size:64;not null;uniqueIndex:idx_permission"`
	CreatedAt int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
}

func (Permission) TableName() string { return "permissions" }

type RolePermission struct {
	RoleId       int   `json:"roleId" gorm:"column:role_id;primaryKey"`
	PermissionId int   `json:"permissionId" gorm:"column:permission_id;primaryKey"`
	CreatedAt    int64 `json:"createdAt" gorm:"autoCreateTime:milli"`
}

func (RolePermission) TableName() string { return "role_permissions" }

type UserRole struct {
	Id        int   `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId    int   `json:"userId" gorm:"column:user_id;uniqueIndex;not null"`
	RoleId    int   `json:"roleId" gorm:"column:role_id;not null"`
	CreatedAt int64 `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt int64 `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}

func (UserRole) TableName() string { return "user_roles" }

type Backup struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string `json:"name" gorm:"column:name;size:256;not null"`
	Description string `json:"description" gorm:"column:description;type:text"`
	FilePath    string `json:"filePath" gorm:"column:file_path;size:512;not null"`
	FileSize    int64  `json:"fileSize" gorm:"column:file_size;default:0"`
	Checksum    string `json:"checksum" gorm:"column:checksum;size:128"`
	Encrypted   bool   `json:"encrypted" gorm:"column:encrypted;default:false"`
	EncryptionMethod string `json:"encryptionMethod" gorm:"column:encryption_method;size:32"`
	Status      string `json:"status" gorm:"column:status;size:32;default:completed"`
	Type        string `json:"type" gorm:"column:type;size:32;default:manual"`
	CreatedBy   int    `json:"createdBy" gorm:"column:created_by"`
	CreatedAt   int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
}

func (Backup) TableName() string { return "backups" }

type Certificate struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	Domain      string `json:"domain" gorm:"column:domain;size:256;not null;uniqueIndex"`
	Issuer      string `json:"issuer" gorm:"column:issuer;size:128"`
	Fingerprint string `json:"fingerprint" gorm:"column:fingerprint;size:128"`
	CertFile    string `json:"certFile" gorm:"column:cert_file;size:512"`
	KeyFile     string `json:"keyFile" gorm:"column:key_file;size:512"`
	NotBefore   int64  `json:"notBefore" gorm:"column:not_before"`
	NotAfter    int64  `json:"notAfter" gorm:"column:not_after"`
	AutoRenew   bool   `json:"autoRenew" gorm:"column:auto_renew;default:false"`
	RenewStatus string `json:"renewStatus" gorm:"column:renew_status;size:32;default:ok"`
	RenewCount  int    `json:"renewCount" gorm:"column:renew_count;default:0"`
	Provider    string `json:"provider" gorm:"column:provider;size:64;default:letsencrypt"`
	CreatedAt   int64  `json:"createdAt" gorm:"autoCreateTime:milli"`
	UpdatedAt   int64  `json:"updatedAt" gorm:"autoUpdateTime:milli"`
}

func (Certificate) TableName() string { return "certificates" }

func DefaultRolePermissions() map[Role][]struct{ Resource, Action string } {
	return map[Role][]struct{ Resource, Action string }{
		RoleAdmin: {
			{"users", "read"}, {"users", "write"}, {"users", "delete"},
			{"inbounds", "read"}, {"inbounds", "write"}, {"inbounds", "delete"},
			{"protocols", "read"}, {"protocols", "write"}, {"protocols", "control"},
			{"services", "read"}, {"services", "write"}, {"services", "control"},
			{"wrappers", "read"}, {"wrappers", "write"},
			{"monitoring", "read"}, {"monitoring", "write"},
			{"settings", "read"}, {"settings", "write"},
			{"backup", "read"}, {"backup", "write"},
			{"audit", "read"}, {"audit", "clear"},
			{"roles", "read"}, {"roles", "write"},
			{"nodes", "read"}, {"nodes", "write"},
			{"clients", "read"}, {"clients", "write"},
			{"certificates", "read"}, {"certificates", "write"},
		},
		RoleOperator: {
			{"inbounds", "read"}, {"inbounds", "write"}, {"inbounds", "delete"},
			{"protocols", "read"}, {"protocols", "write"}, {"protocols", "control"},
			{"services", "read"}, {"services", "write"}, {"services", "control"},
			{"wrappers", "read"}, {"wrappers", "write"},
			{"monitoring", "read"},
			{"nodes", "read"}, {"nodes", "write"},
			{"clients", "read"}, {"clients", "write"},
		},
		RoleViewer: {
			{"inbounds", "read"},
			{"protocols", "read"},
			{"services", "read"},
			{"wrappers", "read"},
			{"monitoring", "read"},
			{"nodes", "read"},
			{"clients", "read"},
			{"audit", "read"},
			{"settings", "read"},
		},
		RoleService: {
			{"protocols", "read"}, {"protocols", "control"},
			{"services", "read"}, {"services", "control"},
			{"monitoring", "read"},
			{"inbounds", "read"},
			{"nodes", "read"},
		},
	}
}

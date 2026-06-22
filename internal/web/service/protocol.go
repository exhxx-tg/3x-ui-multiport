package service

import (
	"encoding/json"

	"github.com/exhxx-tg/3x-ui-multiport/internal/database"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database/model"
	xuiProtocol "github.com/exhxx-tg/3x-ui-multiport/internal/protocol"
	"gorm.io/gorm"
)

type ProtocolService struct {
	db       *gorm.DB
	registry *xuiProtocol.Registry
	manager  *xuiProtocol.Manager
}

func NewProtocolService() *ProtocolService {
	reg := xuiProtocol.Global()
	mgr := xuiProtocol.GlobalManager()
	if mgr == nil {
		reg = xuiProtocol.NewRegistry()
		mgr = xuiProtocol.NewManager(reg)
	}

	return &ProtocolService{
		db:       database.GetDB(),
		registry: reg,
		manager:  mgr,
	}
}

func (s *ProtocolService) ListProtocols() []xuiProtocol.ProtocolInfo {
	return xuiProtocol.AllProtocols
}

func (s *ProtocolService) GetProtocol(id xuiProtocol.ProtocolID) *xuiProtocol.ProtocolInfo {
	return xuiProtocol.GetProtocolInfo(id)
}

func (s *ProtocolService) GetStatus(id xuiProtocol.ProtocolID) (xuiProtocol.Status, error) {
	return s.manager.GetProtocolStatus(id)
}

func (s *ProtocolService) Start(id xuiProtocol.ProtocolID) error {
	return s.manager.StartProtocol(id)
}

func (s *ProtocolService) Stop(id xuiProtocol.ProtocolID) error {
	return s.manager.StopProtocol(id)
}

func (s *ProtocolService) Restart(id xuiProtocol.ProtocolID) error {
	return s.manager.RestartProtocol(id)
}

func (s *ProtocolService) ListServices() ([]model.Service, error) {
	var services []model.Service
	result := s.db.Find(&services)
	return services, result.Error
}

func (s *ProtocolService) GetService(id int) (*model.Service, error) {
	var svc model.Service
	result := s.db.First(&svc, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &svc, nil
}

func (s *ProtocolService) CreateService(svc *model.Service) error {
	return s.db.Create(svc).Error
}

func (s *ProtocolService) UpdateService(svc *model.Service) error {
	return s.db.Save(svc).Error
}

func (s *ProtocolService) DeleteService(id int) error {
	return s.db.Delete(&model.Service{}, id).Error
}

func (s *ProtocolService) ListWrappers() ([]model.TransportWrapper, error) {
	var wrappers []model.TransportWrapper
	result := s.db.Find(&wrappers)
	return wrappers, result.Error
}

func (s *ProtocolService) CreateWrapper(w *model.TransportWrapper) error {
	return s.db.Create(w).Error
}

func (s *ProtocolService) UpdateWrapper(w *model.TransportWrapper) error {
	return s.db.Save(w).Error
}

func (s *ProtocolService) DeleteWrapper(id int) error {
	return s.db.Delete(&model.TransportWrapper{}, id).Error
}

func (s *ProtocolService) GetProtocolConfig(protoName string) (*model.ProtocolConfig, error) {
	var cfg model.ProtocolConfig
	result := s.db.Where("protocol = ?", protoName).First(&cfg)
	if result.Error != nil {
		return nil, result.Error
	}
	return &cfg, nil
}

func (s *ProtocolService) SaveProtocolConfig(protoName string, cfg any) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	existing, err := s.GetProtocolConfig(protoName)
	if err != nil {
		pc := &model.ProtocolConfig{
			Protocol: protoName,
			Config:   string(data),
			Version:  xuiProtocol.EcosystemVersion,
			Enabled:  true,
		}
		return s.db.Create(pc).Error
	}
	return s.db.Model(existing).Updates(map[string]any{
		"config":  string(data),
		"version": xuiProtocol.EcosystemVersion,
	}).Error
}

func (s *ProtocolService) HealthCheck(id xuiProtocol.ProtocolID) error {
	p := s.registry.Get(id)
	if p == nil {
		return xuiProtocol.ErrProtocolNotFound
	}
	if ss, ok := p.(xuiProtocol.StandaloneService); ok {
		return ss.HealthCheck()
	}
	status, _ := s.GetStatus(id)
	if status != xuiProtocol.StatusRunning {
		return xuiProtocol.ErrNotRunning
	}
	return nil
}

func (s *ProtocolService) GetSupportedWrappers(baseProtocol xuiProtocol.ProtocolID) []xuiProtocol.ProtocolID {
	return s.manager.GetSupportedWrappers(baseProtocol)
}

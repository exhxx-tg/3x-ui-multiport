package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/exhxx-tg/3x-ui-multiport/internal/config"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database"
	"github.com/exhxx-tg/3x-ui-multiport/internal/database/model"
)

type BackupService struct{}

func (s *BackupService) CreateBackup(description string, createdBy int, encrypt bool) (*model.Backup, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	backupDir := filepath.Join(".", "backups")
	if err := os.MkdirAll(backupDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create backup dir: %w", err)
	}

	timestamp := time.Now().UnixMilli()
	backupName := fmt.Sprintf("x-ui-backup-%d.sql", timestamp)
	backupPath := filepath.Join(backupDir, backupName)

	dbPath := config.GetDBPath()
	if err := database.DumpSQLite(dbPath, backupPath); err != nil {
		return nil, fmt.Errorf("failed to dump database: %w", err)
	}

	checksum, err := fileSHA256(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to compute checksum: %w", err)
	}

	fileInfo, _ := os.Stat(backupPath)
	fileSize := int64(0)
	if fileInfo != nil {
		fileSize = fileInfo.Size()
	}

	encryptionMethod := ""
	if encrypt {
		encryptionMethod = "aes-256-gcm"
		encKey, err := DeriveEncryptionKey()
		if err != nil {
			return nil, fmt.Errorf("failed to get encryption key: %w", err)
		}
		if err := encryptFile(backupPath, encKey); err != nil {
			return nil, fmt.Errorf("failed to encrypt backup: %w", err)
		}
		backupPath += ".enc"
	}

	backup := &model.Backup{
		Name:             backupName,
		Description:      description,
		FilePath:         backupPath,
		FileSize:         fileSize,
		Checksum:         checksum,
		Encrypted:        encrypt,
		EncryptionMethod: encryptionMethod,
		Status:           "completed",
		Type:             "manual",
		CreatedBy:        createdBy,
		CreatedAt:        time.Now().UnixMilli(),
	}

	if err := db.Create(backup).Error; err != nil {
		return nil, fmt.Errorf("failed to save backup record: %w", err)
	}

	return backup, nil
}

func (s *BackupService) ListBackups() ([]model.Backup, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var backups []model.Backup
	if err := db.Order("id DESC").Find(&backups).Error; err != nil {
		return nil, err
	}
	return backups, nil
}

func (s *BackupService) GetBackup(id int) (*model.Backup, error) {
	db := database.GetDB()
	if db == nil {
		return nil, fmt.Errorf("database not available")
	}

	var backup model.Backup
	if err := db.First(&backup, id).Error; err != nil {
		return nil, err
	}
	return &backup, nil
}

func (s *BackupService) DeleteBackup(id int) error {
	db := database.GetDB()
	if db == nil {
		return fmt.Errorf("database not available")
	}

	backup, err := s.GetBackup(id)
	if err != nil {
		return err
	}

	os.Remove(backup.FilePath)
	return db.Delete(&model.Backup{}, id).Error
}

func (s *BackupService) RestoreBackup(id int) error {
	backup, err := s.GetBackup(id)
	if err != nil {
		return err
	}

	restorePath := backup.FilePath
	if backup.Encrypted {
		decryptedPath := restorePath + ".decrypted"
		defer os.Remove(decryptedPath)

		encKey, err := DeriveEncryptionKey()
		if err != nil {
			return fmt.Errorf("failed to get encryption key: %w", err)
		}
		if err := decryptFile(restorePath, decryptedPath, encKey); err != nil {
			return fmt.Errorf("failed to decrypt backup: %w", err)
		}
		restorePath = decryptedPath
	}

	dbPath := config.GetDBPath()
	tmpPath := dbPath + ".restoring"
	if err := database.RestoreSQLite(restorePath, tmpPath); err != nil {
		return fmt.Errorf("failed to restore database: %w", err)
	}

	origPath := dbPath + ".pre-restore"
	if err := os.Rename(dbPath, origPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to backup current db: %w", err)
	}
	if err := os.Rename(tmpPath, dbPath); err != nil {
		os.Rename(origPath, dbPath)
		return fmt.Errorf("failed to activate restored db: %w", err)
	}

	os.Remove(origPath)
	return nil
}

func (s *BackupService) ExportAuditLogs(format string) (string, []byte, error) {
	db := database.GetDB()
	if db == nil {
		return "", nil, fmt.Errorf("database not available")
	}

	var logs []model.AuditLog
	if err := db.Order("id DESC").Find(&logs).Error; err != nil {
		return "", nil, err
	}

	switch format {
	case "csv":
		data := exportAuditCSV(logs)
		return "audit-export.csv", data, nil
	case "json":
		data := exportAuditJSON(logs)
		return "audit-export.json", data, nil
	default:
		return "", nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func exportAuditCSV(logs []model.AuditLog) []byte {
	var b []byte
	b = append(b, "id,userId,username,action,resource,resourceId,ip,status,createdAt\n"...)
	for _, l := range logs {
		b = append(b, fmt.Sprintf("%d,%d,%s,%s,%s,%s,%s,%s,%d\n",
			l.Id, l.UserId, l.Username, l.Action, l.Resource,
			l.ResourceId, l.Ip, l.Status, l.CreatedAt)...)
	}
	return b
}

func exportAuditJSON(logs []model.AuditLog) []byte {
	data, _ := json.MarshalIndent(logs, "", "  ")
	return data
}

func DeriveEncryptionKey() ([]byte, error) {
	settingSvc := SettingService{}
	secret, err := settingSvc.GetSecret()
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(secret)
	return h[:], nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func encryptFile(path string, key []byte) error {
	plaintext, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return os.WriteFile(path, ciphertext, 0600)
}

func decryptFile(src, dst string, key []byte) error {
	ciphertext, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, plaintext, 0600)
}

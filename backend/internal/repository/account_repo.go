package repository

import (
	"fmt"

	"WeMediaSpider/backend/internal/database/models"

	"gorm.io/gorm"
)

// AccountRepository 公众号仓储接口
type AccountRepository interface {
	Create(account *models.Account) error
	FindByFakeid(fakeid string) (*models.Account, error)
	FindOrCreate(fakeid, name string) (*models.Account, error)
	List() ([]*models.Account, error)
}

// AccountRepositoryImpl 公众号仓储实现
type AccountRepositoryImpl struct {
	db *gorm.DB
}

// NewAccountRepository 创建公众号仓储
func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &AccountRepositoryImpl{db: db}
}

// Create 创建公众号
func (r *AccountRepositoryImpl) Create(account *models.Account) error {
	return r.db.Create(account).Error
}

// FindByFakeid 根据 fakeid 查找公众号
func (r *AccountRepositoryImpl) FindByFakeid(fakeid string) (*models.Account, error) {
	var account models.Account
	err := r.db.Where("fakeid = ?", fakeid).First(&account).Error
	if err != nil {
		return nil, err
	}
	return &account, nil
}

// FindOrCreate 查找或创建公众号
func (r *AccountRepositoryImpl) FindOrCreate(fakeid, name string) (*models.Account, error) {
	var account models.Account
	err := r.db.Where("fakeid = ?", fakeid).FirstOrCreate(&account, models.Account{
		Fakeid: fakeid,
		Name:   name,
	}).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find or create account: %w", err)
	}
	return &account, nil
}

// List 获取所有公众号
func (r *AccountRepositoryImpl) List() ([]*models.Account, error) {
	var accounts []*models.Account
	err := r.db.Order("name ASC").Find(&accounts).Error
	return accounts, err
}

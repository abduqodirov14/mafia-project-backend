package repositories

import (
	"mafia-bot/db/models"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetOrCreate(telegramID int64, username, firstName string) (*models.User, error) {
	var user models.User
	result := r.db.Where("telegram_id = ?", telegramID).First(&user)
	if result.Error == gorm.ErrRecordNotFound {
		user = models.User{
			TelegramID: telegramID,
			Username:   username,
			FirstName:  firstName,
			Coins:      100,
			Level:      1,
		}
		if err := r.db.Create(&user).Error; err != nil {
			return nil, err
		}
	}
	return &user, nil
}

func (r *UserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

func (r *UserRepository) GetTopUsers(limit int) ([]models.User, error) {
	var users []models.User
	err := r.db.Order("xp DESC").Limit(limit).Find(&users).Error
	return users, err
}

func (r *UserRepository) AddCoins(telegramID int64, amount int) error {
	return r.db.Model(&models.User{}).
		Where("telegram_id = ?", telegramID).
		UpdateColumn("coins", gorm.Expr("coins + ?", amount)).Error
}

func (r *UserRepository) HasItem(userID uint, itemID uint) bool {
	var count int64
	r.db.Model(&models.UserItem{}).
		Where("user_id = ? AND item_id = ?", userID, itemID).
		Count(&count)
	return count > 0
}

func (r *UserRepository) BuyItem(userID uint, itemID uint, price int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, userID).Error; err != nil {
			return err
		}
		if user.Coins < price {
			return gorm.ErrInvalidData
		}
		user.Coins -= price
		if err := tx.Save(&user).Error; err != nil {
			return err
		}
		userItem := models.UserItem{UserID: userID, ItemID: itemID}
		return tx.Create(&userItem).Error
	})
}

func (r *UserRepository) GetUserItems(userID uint) ([]models.Item, error) {
	var items []models.Item
	err := r.db.Joins("JOIN user_items ON user_items.item_id = items.id").
		Where("user_items.user_id = ?", userID).
		Find(&items).Error
	return items, err
}

func (r *UserRepository) GetAllUsers() ([]models.User, error) {
	var users []models.User
	err := r.db.Find(&users).Error
	return users, err
}

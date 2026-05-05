package dao

import (
	"errors"

	"geekedu/common/errcode"
	"geekedu/logic-server/model"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type OrderDao struct {
	db *gorm.DB
}

func NewOrderDao() *OrderDao {
	return &OrderDao{db: GetDB()}
}

func NewOrderDaoWithDB(db *gorm.DB) *OrderDao {
	return &OrderDao{db: db}
}

func (d *OrderDao) CreateOrder(order *model.Order) error {
	err := d.db.Create(order).Error
	if isDuplicateKey(err) {
		return errcode.ErrAlreadyPurchased
	}
	return err
}

func (d *OrderDao) CheckPurchase(userID, courseID uint64) (bool, error) {
	var count int64
	err := d.db.Model(&model.Order{}).
		Where("user_id = ? AND course_id = ?", userID, courseID).
		Count(&count).Error
	return count > 0, err
}

func isDuplicateKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

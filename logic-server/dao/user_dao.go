package dao

import (
	"errors"

	"geekedu/common/errcode"
	"geekedu/logic-server/model"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type UserDao struct {
	db *gorm.DB
}

// NewUserDao 默认构造函数（生产环境，连接真实 DB）
func NewUserDao() *UserDao {
	return &UserDao{db: GetDB()}
}

// NewUserDaoWithDB 依赖注入构造函数（测试环境，传入 Mock DB）
func NewUserDaoWithDB(db *gorm.DB) *UserDao {
	return &UserDao{db: db}
}

func (d *UserDao) CreateUser(user *model.User) error {
	err := d.db.Create(user).Error
	if isUserDuplicateKey(err) {
		return errcode.ErrUsernameExists
	}
	return err
}

func (d *UserDao) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	err := d.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func isUserDuplicateKey(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

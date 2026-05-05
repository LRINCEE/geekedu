package service

import (
	"context"
	"regexp"
	"testing"
	"time"

	pb "geekedu/common/pb"
	"geekedu/logic-server/dao"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type fakeTokenProvider struct{}

func (f *fakeTokenProvider) Generate(userID int64, role int32) (string, error) {
	return "test-token", nil
}

// setupMockDB 初始化一个拦截真实 SQL 的 Mock 数据库
func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)

	dialector := mysql.New(mysql.Config{
		Conn:                      mockDB,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	assert.NoError(t, err)

	return db, mock
}

func TestUserService_Login_Success(t *testing.T) {
	// 1. 创建假的 GORM 数据库并注入到 Service
	db, mock := setupMockDB(t)
	userDao := dao.NewUserDaoWithDB(db)
	service := NewUserServiceServer(userDao, NewBcryptPasswordManager(), &fakeTokenProvider{})

	// 2. 预设一个提前被 bcrypt 加密好的密码 "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	// 3. 告诉 sqlmock: 如果看到类似 "SELECT * FROM `users` WHERE username = ?" 的语句，不要连 MySQL
	// 直接返回这行我们捏造的数据给 GORM！
	rows := sqlmock.NewRows([]string{"id", "username", "password", "role", "created_at", "updated_at"}).
		AddRow(1001, "testuser", string(hashedPassword), 1, time.Now(), time.Now())

	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE username = ? ORDER BY `users`.`id` LIMIT ?")).
		WithArgs("testuser", 1).
		WillReturnRows(rows)

	// 4. 发起真实业务请求
	req := &pb.LoginRequest{
		Username: "testuser",
		Password: "password123",
	}

	resp, err := service.Login(context.Background(), req)

	// 5. 验证核心业务逻辑：GORM 能不能查出数据？bcrypt 能不能校验通过？JWT 能不能生成？
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "test-token", resp.Token) // 验证：成功生成了 Token
	assert.Equal(t, int64(1001), resp.UserId) // 验证：正确返回了用户 ID
	assert.Equal(t, int32(1), resp.Role)

	// 6. 验证预设的 SQL 拦截规则是否真的被触发过
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestUserService_Login_EmptyParams(t *testing.T) {
	service := NewUserServiceServer(nil, nil, nil) // 不需要依赖，因为在校验参数时就应该返回

	req := &pb.LoginRequest{Username: "", Password: ""}
	resp, err := service.Login(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid params")
}

func TestUserService_Login_UserNotFound(t *testing.T) {
	db, mock := setupMockDB(t)
	userDao := dao.NewUserDaoWithDB(db)
	service := NewUserServiceServer(userDao, NewBcryptPasswordManager(), &fakeTokenProvider{})

	// 模拟数据库中查不到用户
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE username = ? ORDER BY `users`.`id` LIMIT ?")).
		WithArgs("notexist", 1).
		WillReturnRows(sqlmock.NewRows([]string{})) // 返回空行

	req := &pb.LoginRequest{
		Username: "notexist",
		Password: "password123",
	}

	resp, err := service.Login(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid credential") // 检查错误信息是否符合预期
}

func TestUserService_Login_WrongPassword(t *testing.T) {
	db, mock := setupMockDB(t)
	userDao := dao.NewUserDaoWithDB(db)
	service := NewUserServiceServer(userDao, NewBcryptPasswordManager(), &fakeTokenProvider{})

	// 预设正确的密码哈希 (password123)
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	rows := sqlmock.NewRows([]string{"id", "username", "password", "role", "created_at", "updated_at"}).
		AddRow(1001, "testuser", string(hashedPassword), 1, time.Now(), time.Now())

	// 模拟能查到用户
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE username = ? ORDER BY `users`.`id` LIMIT ?")).
		WithArgs("testuser", 1).
		WillReturnRows(rows)

	// 发起请求，但是使用了错误的密码
	req := &pb.LoginRequest{
		Username: "testuser",
		Password: "wrongpassword", // 故意输入错误密码
	}

	resp, err := service.Login(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid credential")
}

func TestUserService_Register_EmptyParams(t *testing.T) {
	service := NewUserServiceServer(nil, nil, nil)

	req := &pb.RegisterRequest{Username: "", Password: ""}
	resp, err := service.Register(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid params")
}

func TestUserService_Register_Success(t *testing.T) {
	// 1. 创建假的 GORM 数据库并注入到 Service
	db, mock := setupMockDB(t)
	userDao := dao.NewUserDaoWithDB(db)
	service := NewUserServiceServer(userDao, NewBcryptPasswordManager(), &fakeTokenProvider{})

	// 2. 告诉 sqlmock 预期的行为
	// 2.1 首先会查询用户是否存在
	mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `users` WHERE username = ? ORDER BY `users`.`id` LIMIT ?")).
		WithArgs("newuser", 1).
		WillReturnRows(sqlmock.NewRows([]string{})) // 返回空，表示不存在

	// 2.2 然后执行插入操作
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `users`")).
		WithArgs("newuser", sqlmock.AnyArg(), 0, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1002, 1))
	mock.ExpectCommit()

	// 3. 发起真实业务请求
	req := &pb.RegisterRequest{
		Username: "newuser",
		Password: "password123",
	}

	resp, err := service.Register(context.Background(), req)

	// 4. 验证
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(1002), resp.UserId)

	// 5. 验证预设的 SQL 拦截规则是否真的被触发过
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

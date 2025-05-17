package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"online-server/handler"
	"online-server/model"
	"online-server/utils"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 测试用的数据库连接和gin引擎
var (
	testDB     *gorm.DB
	testRouter *gin.Engine
)

// 测试用的用户数据
var (
	adminUser = model.User{
		Username: "admin",
		Email:    "admin@example.com",
		Role:     model.RoleAdmin,
	}
	officerUser = model.User{
		Username: "officer",
		Email:    "officer@example.com",
		Role:     model.RoleOfficer,
	}
	guestUser = model.User{
		Username: "guest",
		Email:    "guest@example.com",
		Role:     model.RoleGuest,
	}
	testPassword = "password123"
)

// 测试用的 Token
var (
	adminToken   string
	officerToken string
	guestToken   string
	invalidToken = "invalid.token.value"
)

// 初始化测试环境
func setupTestEnv(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建测试数据库连接
	var err error
	testDB, err = gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// 迁移数据库架构
	err = testDB.AutoMigrate(&model.User{})
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	// 设置为全局数据库连接
	utils.SetDB(testDB)

	// 设置JWT密钥
	utils.SetJWTKey([]byte("test-jwt-key"))

	// 创建测试用户
	hashedPassword, err := utils.HashPassword(testPassword)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}
	adminUser.Password = hashedPassword
	officerUser.Password = hashedPassword
	guestUser.Password = hashedPassword

	// 保存到数据库
	if err := testDB.Create(&adminUser).Error; err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}
	if err := testDB.Create(&officerUser).Error; err != nil {
		t.Fatalf("Failed to create officer user: %v", err)
	}
	if err := testDB.Create(&guestUser).Error; err != nil {
		t.Fatalf("Failed to create guest user: %v", err)
	}

	// 生成令牌
	var tokenErr error
	adminToken, tokenErr = utils.GenerateToken(adminUser.Username, string(adminUser.Role), time.Hour)
	if tokenErr != nil {
		t.Fatalf("Failed to generate admin token: %v", tokenErr)
	}

	officerToken, tokenErr = utils.GenerateToken(officerUser.Username, string(officerUser.Role), time.Hour)
	if tokenErr != nil {
		t.Fatalf("Failed to generate officer token: %v", tokenErr)
	}

	guestToken, tokenErr = utils.GenerateToken(guestUser.Username, string(guestUser.Role), time.Hour)
	if tokenErr != nil {
		t.Fatalf("Failed to generate guest token: %v", tokenErr)
	}

	// 创建测试路由器
	testRouter = gin.New()
	testRouter.Use(gin.Recovery())

	// 公开路由（无需认证）
	public := testRouter.Group("/api")
	{
		public.POST("/login", handler.Login)
		public.POST("/register", handler.Register)
		public.POST("/check-auth", handler.CheckAuth)
	}

	// 用户路由（需认证）
	users := testRouter.Group("/api/users")
	users.Use(func(c *gin.Context) {
		// 简化的JWT认证中间件（仅用于测试）
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "需要登录"})
			c.Abort()
			return
		}

		var username, role string
		var err error

		switch token {
		case adminToken:
			username = adminUser.Username
			role = string(adminUser.Role)
		case officerToken:
			username = officerUser.Username
			role = string(officerUser.Role)
		case guestToken:
			username = guestUser.Username
			role = string(guestUser.Role)
		default:
			err = fmt.Errorf("无效令牌")
		}

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "令牌无效"})
			c.Abort()
			return
		}

		// 设置用户信息到上下文
		c.Set("Username", username)
		c.Set("Role", role)
		c.Set("UserID", getUserIDByUsername(username))

		c.Next()
	})
	{
		users.GET("/profile", handler.GetCurrentUser)
		users.POST("/logout", handler.Logout)
		users.POST("/change-password", handler.ChangePassword)

		// 管理员功能
		admin := users.Group("/admin")
		admin.Use(func(c *gin.Context) {
			// 简化的管理员权限检查中间件（仅用于测试）
			role, exists := c.Get("Role")
			if !exists || role.(string) != string(model.RoleAdmin) {
				c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "权限不足"})
				c.Abort()
				return
			}
			c.Next()
		})
		{
			admin.GET("/users", handler.GetUsers)
			admin.GET("/users/:id", handler.GetUserByID)
			admin.PUT("/users/:id/role", handler.UpdateUserRole)
			admin.PUT("/users/:id/username", handler.UpdateUserID)
			admin.DELETE("/users/:id", handler.DeleteUser)
		}
	}
}

// 测试清理函数
func teardownTestEnv(t *testing.T) {
	// 清理数据库
	sqlDB, err := testDB.DB()
	if err != nil {
		t.Errorf("Failed to get DB instance: %v", err)
		return
	}
	sqlDB.Close()
}

// 辅助函数：通过用户名获取用户ID
func getUserIDByUsername(username string) uint {
	var user model.User
	err := testDB.Where("username = ?", username).First(&user).Error
	if err != nil {
		log.Printf("Error finding user by username: %v", err)
		return 0
	}
	return user.ID
}

// 测试用户注册
func TestRegister(t *testing.T) {
	setupTestEnv(t)
	defer teardownTestEnv(t)

	// 测试场景：成功注册
	t.Run("Success", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "newuser",
			"password": "password123",
			"email":    "newuser@example.com",
		}
		jsonData, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)

		assert.Equal(t, 200, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])
		assert.Equal(t, "注册成功", response["message"])

		// 验证用户已保存到数据库
		var user model.User
		err = testDB.Where("username = ?", "newuser").First(&user).Error
		assert.NoError(t, err)
		assert.Equal(t, "newuser", user.Username)
		assert.Equal(t, "newuser@example.com", user.Email)
	})

	// 测试场景：用户名已存在
	t.Run("UserExists", func(t *testing.T) {
		reqBody := map[string]string{
			"username": adminUser.Username, // 使用已存在的用户名
			"password": "password123",
			"email":    "unique@example.com",
		}
		jsonData, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)

		assert.Equal(t, 400, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(400), response["code"])
		assert.Contains(t, response["message"], "用户名已存在")
	})

	// 测试场景：邮箱已被使用
	t.Run("EmailExists", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "uniqueuser",
			"password": "password123",
			"email":    adminUser.Email, // 使用已存在的邮箱
		}
		jsonData, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/register", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)

		assert.Equal(t, 400, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(400), response["code"])
		assert.Contains(t, response["message"], "邮箱已被使用")
	})
}

// 测试用户登录
func TestLogin(t *testing.T) {
	setupTestEnv(t)
	defer teardownTestEnv(t)

	// 测试场景：成功登录
	t.Run("Success", func(t *testing.T) {
		reqBody := map[string]string{
			"username": adminUser.Username,
			"password": testPassword,
		}
		jsonData, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)

		assert.Equal(t, 200, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])
		assert.Equal(t, "登录成功", response["message"])

		// 验证返回了token和用户信息
		data := response["data"].(map[string]interface{})
		assert.NotEmpty(t, data["token"])
		user := data["user"].(map[string]interface{})
		assert.Equal(t, adminUser.Username, user["username"])
		assert.Equal(t, string(adminUser.Role), user["role"])
	})

	// 测试场景：密码错误
	t.Run("WrongPassword", func(t *testing.T) {
		reqBody := map[string]string{
			"username": adminUser.Username,
			"password": "wrongpassword",
		}
		jsonData, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)

		assert.Equal(t, 401, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(401), response["code"])
		assert.Contains(t, response["message"], "用户名或密码错误")
	})

	// 测试场景：用户不存在
	t.Run("UserNotFound", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "nonexistentuser",
			"password": testPassword,
		}
		jsonData, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)

		assert.Equal(t, 401, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(401), response["code"])
		assert.Contains(t, response["message"], "用户名或密码错误")
	})
}

// 测试Token验证
func TestCheckAuth(t *testing.T) {
	setupTestEnv(t)
	defer teardownTestEnv(t)

	// 测试场景：有效令牌
	t.Run("ValidToken", func(t *testing.T) {
		reqBody := map[string]string{
			"token": adminToken,
		}
		jsonData, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/check-auth", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)

		assert.Equal(t, 200, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(200), response["code"])
		assert.Equal(t, "令牌有效", response["message"])
		assert.Equal(t, true, response["valid"])

		// 验证返回了用户信息
		data := response["data"].(map[string]interface{})
		user := data["user"].(map[string]interface{})
		assert.Equal(t, adminUser.Username, user["username"])
		assert.Equal(t, string(adminUser.Role), user["role"])
	})

	// 测试场景：无效令牌
	t.Run("InvalidToken", func(t *testing.T) {
		reqBody := map[string]string{
			"token": invalidToken,
		}
		jsonData, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/check-auth", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)

		assert.Equal(t, 401, resp.Code)

		var response map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(401), response["code"])
		assert.Equal(t, false, response["valid"])
	})
}

// 测试获取当前用户信息
func TestGetCurrentUser(t *testing.T) {
	setupTestEnv(t)
	defer teardownTestEnv(t)

	req, _ := http.NewRequest("GET", "/api/users/profile", nil)
	req.Header.Set("Authorization", adminToken)
	resp := httptest.NewRecorder()
	testRouter.ServeHTTP(resp, req)

	assert.Equal(t, 200, resp.Code)

	var response map[string]interface{}
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(200), response["code"])
	assert.Equal(t, "获取当前用户信息成功", response["message"])

	data := response["data"].(map[string]interface{})
	assert.Equal(t, adminUser.Username, data["username"])
	assert.Equal(t, string(adminUser.Role), data["role"])
}

// 测试更新用户角色
func TestUpdateUserRole(t *testing.T) {
	setupTestEnv(t)
	defer teardownTestEnv(t)

	// 测试场景：管理员成功更新用户角色
	t.Run("AdminSuccess", func(t *testing.T) {
		reqBody := map[string]string{
			"role": string(model.RoleOfficer),
		}
		jsonData, _ := json.Marshal(reqBody)

		url := fmt.Sprintf("/api/users/admin/users/%d/role", guestUser.ID)
		req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", adminToken)
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)

		assert.Equal(t, 200, resp.Code)

		// 验证角色已更新
		var updatedUser model.User
		testDB.First(&updatedUser, guestUser.ID)
		assert.Equal(t, model.RoleOfficer, updatedUser.Role)
	})

	// 测试场景：尝试更新管理员角色
	t.Run("UpdateAdmin", func(t *testing.T) {
		reqBody := map[string]string{
			"role": string(model.RoleGuest),
		}
		jsonData, _ := json.Marshal(reqBody)

		url := fmt.Sprintf("/api/users/admin/users/%d/role", adminUser.ID)
		req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", adminToken)
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)

		assert.NotEqual(t, 200, resp.Code)

		// 验证角色未更新
		var updatedUser model.User
		testDB.First(&updatedUser, adminUser.ID)
		assert.Equal(t, model.RoleAdmin, updatedUser.Role)
	})

	// 测试场景：非管理员尝试更新角色
	t.Run("NonAdmin", func(t *testing.T) {
		reqBody := map[string]string{
			"role": string(model.RoleGuest),
		}
		jsonData, _ := json.Marshal(reqBody)

		url := fmt.Sprintf("/api/users/admin/users/%d/role", guestUser.ID)
		req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", officerToken) // 使用警员令牌
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)

		assert.Equal(t, 403, resp.Code) // 权限不足
	})
}

// 测试更新用户名
func TestUpdateUserID(t *testing.T) {
	setupTestEnv(t)
	defer teardownTestEnv(t)

	// 测试场景：管理员成功更新用户名
	t.Run("AdminSuccess", func(t *testing.T) {
		newUsername := "updated_guest"
		reqBody := map[string]string{
			"username": newUsername,
		}
		jsonData, _ := json.Marshal(reqBody)

		url := fmt.Sprintf("/api/users/admin/users/%d/username", guestUser.ID)
		req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", adminToken)
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)

		assert.Equal(t, 200, resp.Code)

		// 验证用户名已更新
		var updatedUser model.User
		testDB.First(&updatedUser, guestUser.ID)
		assert.Equal(t, newUsername, updatedUser.Username)
	})

	// 测试场景：尝试使用已存在的用户名
	t.Run("DuplicateUsername", func(t *testing.T) {
		reqBody := map[string]string{
			"username": officerUser.Username, // 使用已存在的用户名
		}
		jsonData, _ := json.Marshal(reqBody)

		url := fmt.Sprintf("/api/users/admin/users/%d/username", guestUser.ID)
		req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", adminToken)
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)

		assert.Equal(t, 400, resp.Code) // 业务逻辑错误
	})

	// 测试场景：尝试更新管理员用户名
	t.Run("UpdateAdmin", func(t *testing.T) {
		reqBody := map[string]string{
			"username": "new_admin_name",
		}
		jsonData, _ := json.Marshal(reqBody)

		url := fmt.Sprintf("/api/users/admin/users/%d/username", adminUser.ID)
		req, _ := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", adminToken)
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)

		assert.NotEqual(t, 200, resp.Code)

		// 验证用户名未更新
		var updatedUser model.User
		testDB.First(&updatedUser, adminUser.ID)
		assert.Equal(t, adminUser.Username, updatedUser.Username)
	})
}

// 测试删除用户
func TestDeleteUser(t *testing.T) {
	setupTestEnv(t)
	defer teardownTestEnv(t)

	// 创建一个用于删除的测试用户
	deleteUser := model.User{
		Username: "deleteuser",
		Password: "password",
		Email:    "delete@example.com",
		Role:     model.RoleGuest,
	}
	testDB.Create(&deleteUser)

	// 测试场景：成功删除用户
	t.Run("Success", func(t *testing.T) {
		url := fmt.Sprintf("/api/users/admin/users/%d", deleteUser.ID)
		req, _ := http.NewRequest("DELETE", url, nil)
		req.Header.Set("Authorization", adminToken)
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)

		assert.Equal(t, 200, resp.Code)

		// 验证用户已删除
		var count int64
		testDB.Model(&model.User{}).Where("id = ?", deleteUser.ID).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	// 测试场景：非管理员尝试删除用户
	t.Run("NonAdmin", func(t *testing.T) {
		// 创建另一个用于测试的用户
		anotherUser := model.User{
			Username: "anotheruser",
			Password: "password",
			Email:    "another@example.com",
			Role:     model.RoleGuest,
		}
		testDB.Create(&anotherUser)

		url := fmt.Sprintf("/api/users/admin/users/%d", anotherUser.ID)
		req, _ := http.NewRequest("DELETE", url, nil)
		req.Header.Set("Authorization", guestToken) // 使用游客令牌
		resp := httptest.NewRecorder()
		testRouter.ServeHTTP(resp, req)

		assert.Equal(t, 403, resp.Code) // 权限不足

		// 验证用户未删除
		var count int64
		testDB.Model(&model.User{}).Where("id = ?", anotherUser.ID).Count(&count)
		assert.Equal(t, int64(1), count)
	})
}

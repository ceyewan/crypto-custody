package handlers

import (
	"errors"
	"log"
	"net/http"

	"backend/config"
	"backend/models"

	//"backend/servers"

	"backend/utils"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

func Login(c *gin.Context) {
	var loginData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	// 绑定 JSON 数据到结构体
	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数不正确"})
		return
	}
	var users models.User
	if err := config.DB.Where("Username = ?", loginData.Username).First(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !utils.CheckPasswordHash(loginData.Password, users.Password) {
		err := errors.New("密码错误")
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	token, err := utils.GenerateJWT(users.Userid, users.Username, users.Roleid)

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"token": token})
}

func Register(c *gin.Context) {
	var registerData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	// 绑定 JSON 数据到结构体
	if err := c.ShouldBindJSON(&registerData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数不正确"})
		return
	}
	var users models.User
	if err := config.DB.Where("Username = ?", registerData.Username).First(&users).Error; err == nil {
		err := errors.New("用户名已存在")
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// 处理其他错误
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	hashedPassword, err := utils.HashPassword(registerData.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
	}

	// 创建用户
	user := models.User{
		Username: registerData.Username,
		Password: hashedPassword,
		Roleid:   4,
	}
	if err := config.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

func GetUsers(c *gin.Context) {
	var users []models.User
	result := config.DB.Find(&users)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	// 输出 accounts
	log.Println(users)
	c.JSON(http.StatusOK, users)

}

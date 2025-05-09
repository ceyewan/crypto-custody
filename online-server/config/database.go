package config

import (
	"backend/models"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase() {
	dsn := "ceyewan:1027mysql@tcp(127.0.0.1:3306)/wallet_demo?charset=utf8mb4&parseTime=True&loc=Local"
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		panic("Could not connect to the database")
	}

	// 自动迁移
	DB.AutoMigrate(&models.Account{})
	DB.AutoMigrate(&models.User{})
}

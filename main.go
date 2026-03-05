package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gei-git/shortlink/internal/config"
	"github.com/gei-git/shortlink/models"
	"github.com/gei-git/shortlink/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func main() {
	cfg := config.LoadConfig()

	// 连接 PostgreSQL
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 自动建表（开发阶段很方便）
	err = DB.AutoMigrate(&models.ShortLink{}, &models.ClickLog{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	log.Println("✅ Database connected and migrated successfully!")

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"database":  "connected",
			"shortlink": "ready",
		})
	})

	r.POST("/api/shorten", func(c *gin.Context) {
		var req struct {
			URL string `json:"url" binding:"required"`
		}
		// c.ShouldBindJSON(&req)：从请求 body 中解析 JSON，并绑定到 req 结构体
		// 如果成功：req.URL 被赋值为 JSON 中的 "url" 值（例如 "https://example.com"）。
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "需要提供 url 参数"})
			return
		}

		shortCode := utils.GenerateShortCode()

		shortLink := models.ShortLink{
			ShortCode:   shortCode,
			OriginalURL: req.URL,
			ExpiresAt:   nil,
		}

		if err := DB.Create(&shortLink).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建短链接失败"})
			return
		}

		shortUrl := cfg.AppDomain + "/" + shortCode

		// TODO: 实现短链接生成逻辑
		c.JSON(http.StatusOK, gin.H{
			"short_code":   shortCode,
			"short_url":    shortUrl,
			"original_url": req.URL,
		})
	})

	log.Println("🚀 Shortlink API started on http://localhost:8080")
	r.Run(":8080")
}

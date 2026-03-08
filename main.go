package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gei-git/shortlink/internal/config"
	"github.com/gei-git/shortlink/models"
	"github.com/gei-git/shortlink/utils"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	DB  *gorm.DB
	RDB *redis.Client
	cfg *config.Config
	ctx = context.Background()
)

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

	// === Redis 连接 ===
	RDB = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: "",
		DB:       0,
	})
	if err := RDB.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("✅ Redis connected successfully!")

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

	// 短链接跳转（核心！）
	r.GET("/:shortCode", func(c *gin.Context) {
		shortCode := c.Param("shortCode")

		// 1. 先查 Redis 缓存（最快路径）
		if url, err := RDB.Get(ctx, "short:"+shortCode).Result(); err == nil {
			// 命中缓存，直接跳转
			c.Redirect(http.StatusFound, url)
			return
		}

		// 2. 缓存未命中 → 查数据库（回源）
		var shortLink models.ShortLink
		if err := DB.Where("short_code = ?", shortCode).First(&shortLink).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "短链接不存在"})
			return
		}

		// 简单过期检查（后面可扩展）
		if shortLink.ExpiresAt != nil && time.Now().After(*shortLink.ExpiresAt) {
			c.JSON(http.StatusGone, gin.H{"error": "短链接已过期"})
			return
		}

		// 3. 写入 Redis 缓存（默认 1 小时，可调整）
		RDB.Set(ctx, "short:"+shortCode, shortLink.OriginalURL, 1*time.Hour)

		// 4. 增加点击量（后面会改成 Kafka 异步）
		// 异步增加点击量（生产可用 goroutine + Kafka，这里先同步）
		shortLink.ClickCount++
		DB.Save(&shortLink)

		// 5. 302 重定向
		c.Redirect(http.StatusFound, shortLink.OriginalURL)
	})

	log.Println("🚀 Shortlink API started on http://localhost:8080")
	r.Run(":8080")
}

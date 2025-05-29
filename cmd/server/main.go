package main

import (
	"context"
	//"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"expense_tracker/internal/config"
	"expense_tracker/internal/handler"
	"expense_tracker/internal/middleware"
	"expense_tracker/internal/repository"
	"expense_tracker/internal/service"
	"expense_tracker/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading, relying on environment variables")
	}

	// --- Configuration ---
	dbCfg, err := config.LoadDBConfig()
	if err != nil {
		log.Fatalf("Failed to load DB config: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		log.Fatalf("JWT_SECRET_KEY not set in environment")
	}
	jwtExpHoursStr := os.Getenv("JWT_EXPIRATION_HOURS")
	jwtExpHours, err := strconv.ParseInt(jwtExpHoursStr, 10, 64)
	if err != nil {
		log.Printf("Invalid JWT_EXPIRATION_HOURS, defaulting to 24: %v", err)
		jwtExpHours = 24
	}

	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080" // Default port
	}

	uploadsDir := os.Getenv("UPLOADS_DIR")
	if uploadsDir == "" {
		uploadsDir = "uploads" // Default uploads directory
	}
	// Ensure uploads directory exists
	if err := os.MkdirAll(uploadsDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create uploads directory %s: %v", uploadsDir, err)
	}
	log.Printf("Uploads will be stored in: %s", uploadsDir)

	// --- Database Connection ---
	dbPool, err := config.ConnectDB(dbCfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// --- Auto Migration ---
	if err := config.AutoMigrate(dbPool); err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}

	// --- Initialize Utilities ---
	jwtUtil := utils.NewJWTUtil(jwtSecret, jwtExpHours)

	// --- Initialize Repositories ---
	userRepo := repository.NewUserRepository(dbPool)
	transactionRepo := repository.NewTransactionRepository(dbPool)

	// --- Initialize Services ---
	authService := service.NewAuthService(userRepo, jwtUtil)
	transactionService := service.NewTransactionService(transactionRepo, uploadsDir)

	// --- Initialize Handlers ---
	authHandler := handler.NewAuthHandler(authService)
	transactionHandler := handler.NewTransactionHandler(transactionService, uploadsDir)

	// --- Setup Gin Router ---
	// gin.SetMode(gin.ReleaseMode) // Uncomment for production
	router := gin.Default()

	// Simple CORS middleware (allow all for development)
	// For production, configure specific origins, methods, headers
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// --- Initialize Middlewares ---
	jwtAuthMW := middleware.JWTAuthMiddleware(jwtUtil)
	adminRoleMW := middleware.AdminMiddleware()
	// userRoleMW := middleware.UserMiddleware() // Not strictly needed if JWTAuthMW is enough for "logged in"

	// --- Register Routes ---
	apiGroup := router.Group("/api/v1") // Base path for API
	authHandler.RegisterAuthRoutes(apiGroup)
	transactionHandler.RegisterTransactionRoutes(apiGroup, jwtAuthMW, nil /*userRoleMW*/, adminRoleMW)

	// Health check endpoint (not in TZ, but good practice)
	router.GET("/health", func(c *gin.Context) {
		// Check DB connection
		if err := dbPool.Ping(context.Background()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "error", "db": "unhealthy"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok", "db": "healthy"})
	})

	// --- Start Server ---
	srv := &http.Server{
		Addr:    ":" + serverPort,
		Handler: router,
	}

	go func() {
		log.Printf("Server starting on port %s", serverPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

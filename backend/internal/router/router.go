package router

import (
	"github.com/Furkanturan8/goftr-template/config"
	"github.com/Furkanturan8/goftr-template/internal/handler"
	"github.com/Furkanturan8/goftr-template/internal/middleware"
	"github.com/Furkanturan8/goftr-template/internal/repository"
	"github.com/Furkanturan8/goftr-template/internal/service"
	"github.com/Furkanturan8/goftr-template/pkg/email"
	"github.com/Furkanturan8/goftr-template/pkg/monitoring"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/uptrace/bun"
	"time"
)

type Router struct {
	app *fiber.App
	db  *bun.DB
	cfg *config.Config
}

var prometheusEndpoint string
var prometheusEnabled bool

func NewRouter(db *bun.DB, cfg *config.Config) *Router {
	prometheusEnabled = cfg.MonitoringConfig.Prometheus.Enabled
	prometheusEndpoint = cfg.MonitoringConfig.Prometheus.Endpoint

	return &Router{
		app: fiber.New(),
		db:  db,
		cfg: cfg,
	}
}

func (r *Router) SetupRoutes() {
	// Prometheus'un topladığı metrikleri görüntülemek için /metrics endpoint'i
	if prometheusEnabled {
		r.app.Get(prometheusEndpoint, monitoring.MetricsHandler())
	}

	// Middleware'leri ekle
	r.app.Use(logger.New())
	r.app.Use(recover.New())
	r.app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:63342,http://localhost:3005,http://localhost:5173",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Content-Type, Authorization",
	}))

	// Rate limiting middleware'i ekle (30 sn de 10 istek olsun)
	r.app.Use(limiter.New(limiter.Config{
		Max:        10,               // Maksimum istek sayısı
		Expiration: 30 * time.Second, // Zaman aralığı
		KeyGenerator: func(c *fiber.Ctx) string {
			// /metrics endpoint'i için rate limiting'i devre dışı bırak
			if c.Path() == prometheusEndpoint {
				return "metrics_no_limit"
			}
			// Her route'u ayrı ayrı sınırla (örneğin: "/users", "/users/:id", "/auth/login")
			return c.IP() + ":" + c.Path()
		},
	}))

	// Prometheus Middleware ekleyelim
	r.app.Use(monitoring.PrometheusMiddleware())

	// API versiyonu
	api := r.app.Group("/api")
	v1 := api.Group("/v1")

	// Dış paketler emailPkg
	emailPkg := email.NewEmail(
		r.cfg.MailConfig.FromEmail,
		r.cfg.MailConfig.SMTPPassword,
		r.cfg.MailConfig.SMTPHost,
		r.cfg.MailConfig.SMTPPort,
	)

	// Repository'ler
	userRepo := repository.NewUserRepository(r.db)
	authRepo := repository.NewAuthRepository(r.db)

	// Service'ler
	authService := service.NewAuthService(authRepo, userRepo)
	userService := service.NewUserService(userRepo)

	// Handler'lar
	authHandler := handler.NewAuthHandler(authService, emailPkg)
	userHandler := handler.NewUserHandler(userService)

	// Auth routes
	auth := v1.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.RefreshToken)
	auth.Post("/forgot-password", authHandler.ForgotPassword)
	auth.Post("/reset-password", authHandler.ResetPassword)
	auth.Post("/logout", middleware.AuthMiddleware(), authHandler.Logout)

	// User routes - Base group
	users := v1.Group("/users")

	// Normal user routes (profil yönetimi)
	userProfile := users.Group("/me")
	userProfile.Use(middleware.AuthMiddleware()) // Sadece authentication gerekli
	userProfile.Get("/", userHandler.GetProfile)
	userProfile.Put("/", userHandler.UpdateProfile)

	// Admin only routes
	adminUsers := users.Group("/")
	adminUsers.Use(middleware.AuthMiddleware(), middleware.AdminOnly()) // Admin yetkisi gerekli
	adminUsers.Post("/", userHandler.Create)
	adminUsers.Get("/", userHandler.List)
	adminUsers.Get("/:id", userHandler.GetByID)
	adminUsers.Put("/:id", userHandler.Update)
	adminUsers.Delete("/:id", userHandler.Delete)

	// Diğer route grupları buraya eklenecek
}

func (r *Router) GetApp() *fiber.App {
	return r.app
}

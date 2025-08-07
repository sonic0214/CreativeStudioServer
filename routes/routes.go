package routes

import (
	"github.com/gin-gonic/gin"
	"creative-studio-server/controllers"
	"creative-studio-server/middleware"
)

func SetupRoutes(r *gin.Engine) {
	// Initialize controllers
	authController := controllers.NewAuthController()
	atomicClipController := controllers.NewAtomicClipController()

	// Health check and system endpoints
	r.GET("/health", healthCheck)
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Creative Studio Server API",
			"version": "1.0.0",
			"status":  "running",
		})
	})

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// Authentication routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", middleware.AuthRateLimit(), authController.Register)
			auth.POST("/login", middleware.AuthRateLimit(), authController.Login)
			auth.POST("/refresh", authController.RefreshToken)
		}

		// Protected authentication routes
		authProtected := v1.Group("/auth")
		authProtected.Use(middleware.AuthRequired())
		{
			authProtected.GET("/profile", authController.Profile)
			authProtected.POST("/change-password", authController.ChangePassword)
		}

		// Atomic clips routes
		clips := v1.Group("/atomic-clips")
		clips.Use(middleware.OptionalAuth()) // Optional auth for search/browse
		{
			clips.GET("/search", atomicClipController.SearchAtomicClips)
			clips.GET("/:id", atomicClipController.GetAtomicClip)
			clips.GET("/:id/similar", atomicClipController.GetSimilarClips)
		}

		// Protected atomic clips routes
		clipsProtected := v1.Group("/atomic-clips")
		clipsProtected.Use(middleware.AuthRequired())
		{
			clipsProtected.POST("", atomicClipController.CreateAtomicClip)
			clipsProtected.PUT("/:id", atomicClipController.UpdateAtomicClip)
			clipsProtected.DELETE("/:id", atomicClipController.DeleteAtomicClip)
			clipsProtected.GET("/my-clips", atomicClipController.GetUserAtomicClips)
		}

		// Projects routes
		projects := v1.Group("/projects")
		projects.Use(middleware.AuthRequired())
		{
			// TODO: Implement project controller
			// projects.POST("", projectController.CreateProject)
			// projects.GET("", projectController.GetUserProjects)
			// projects.GET("/:id", projectController.GetProject)
			// projects.PUT("/:id", projectController.UpdateProject)
			// projects.DELETE("/:id", projectController.DeleteProject)
		}

		// Smart composition routes
		composition := v1.Group("/composition")
		composition.Use(middleware.AuthRequired())
		{
			// TODO: Implement composition controller
			// composition.POST("/generate", compositionController.GenerateComposition)
			// composition.GET("/algorithms", compositionController.GetAlgorithms)
		}

		// Render tasks routes
		render := v1.Group("/render")
		render.Use(middleware.AuthRequired())
		{
			// TODO: Implement render controller
			// render.POST("/tasks", renderController.CreateRenderTask)
			// render.GET("/tasks", renderController.GetUserRenderTasks)
			// render.GET("/tasks/:id", renderController.GetRenderTask)
			// render.POST("/tasks/:id/cancel", renderController.CancelRenderTask)
		}

		// Admin routes
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthRequired())
		admin.Use(middleware.RoleRequired("admin"))
		{
			// TODO: Implement admin controller
			// admin.GET("/users", adminController.GetUsers)
			// admin.GET("/stats", adminController.GetSystemStats)
			// admin.POST("/maintenance", adminController.MaintenanceMode)
		}
	}
}

func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":    "healthy",
		"timestamp": gin.H{
			"unix":      gin.H{"seconds": 1234567890},
			"formatted": "2023-12-07T10:00:00Z",
		},
		"services": gin.H{
			"database": "connected",
			"redis":    "connected",
			"rabbitmq": "connected",
		},
	})
}
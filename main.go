package main

import (
	"balanca/internal/config"
	"balanca/internal/database"
	"balanca/internal/handlers"
	"balanca/internal/middleware"
	"balanca/internal/repositories"
	"balanca/internal/services"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize database
	if err := database.Connect(&cfg.Database); err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate database
	// if err := database.AutoMigrate(); err != nil {
	// 	log.Fatal("Failed to migrate database:", err)
	// }

	// Initialize repositories
	db := database.GetDB()
	userRepo := repositories.NewUserRepository(db)
	groupRepo := repositories.NewGroupRepository(db)
	transactionRepo := repositories.NewTransactionRepository(db)
	expenseRepo := repositories.NewPlannedExpenseRepository(db)
	auditRepo := repositories.NewAuditLogRepository(db)

	// Initialize services
	authService := services.NewAuthService(userRepo, cfg.JWT.Secret, cfg.JWT.Expiration, cfg.JWT.RefreshTokenExpiration)
	userService := services.NewUserService(userRepo, groupRepo)
	groupService := services.NewGroupService(groupRepo, userRepo, auditRepo, db)
	transactionService := services.NewTransactionService(transactionRepo, userRepo, groupRepo, expenseRepo, auditRepo, db)
	expenseService := services.NewPlannedExpenseService(expenseRepo, userRepo, groupRepo, auditRepo, db)
	reportService := services.NewReportService(transactionRepo, userRepo, groupRepo)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
	groupHandler := handlers.NewGroupHandler(groupService)
	transactionHandler := handlers.NewTransactionHandler(transactionService)
	expenseHandler := handlers.NewPlannedExpenseHandler(expenseService)
	reportHandler := handlers.NewReportHandler(reportService)

	// Setup Gin router
	router := gin.Default()

	// Middleware
	router.Use(middleware.CORS())
	router.Use(middleware.Logger())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "BALANCA API is running",
		})
	})

	// Public routes
	public := router.Group("/api/v1")
	{
		public.POST("/auth/register", authHandler.Register)
		public.POST("/auth/login", authHandler.Login)
		public.POST("/auth/refresh", authHandler.RefreshToken)
	}

	// Protected routes
	protected := router.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	{
		// Auth
		protected.POST("/auth/logout", authHandler.Logout)

		// User
		protected.GET("/users/profile", userHandler.GetProfile)
		protected.PUT("/users/profile", userHandler.UpdateProfile)
		protected.PUT("/users/password", userHandler.ChangePassword)
		protected.GET("/users/search", userHandler.SearchUsers)
		protected.GET("/users/groups", userHandler.GetUserGroups)

		// Group
		protected.POST("/groups", groupHandler.CreateGroup)
		protected.GET("/groups", groupHandler.GetGroups)
		protected.GET("/groups/:groupId", groupHandler.GetGroup)
		protected.POST("/groups/:groupId/invite", groupHandler.InviteMember)
		protected.POST("/invitations/:invitationId/accept", groupHandler.AcceptInvitation)
		protected.POST("/invitations/:invitationId/reject", groupHandler.RejectInvitation)
		protected.PUT("/groups/:groupId/members/role", groupHandler.UpdateMemberRole)
		protected.DELETE("/groups/:groupId/members/:userId", groupHandler.RemoveMember)
		protected.GET("/invitations/pending", groupHandler.GetPendingInvitations)
		protected.POST("/groups/:groupId/leave", groupHandler.LeaveGroup)
		protected.DELETE("/groups/:groupId", groupHandler.DeleteGroup)

		// Personal Transactions
		protected.POST("/transactions/personal", transactionHandler.CreatePersonalTransaction)
		protected.GET("/transactions/personal", transactionHandler.GetPersonalTransactions)
		protected.GET("/transactions/:transactionId", transactionHandler.GetTransaction)

		// Group Transactions
		protected.POST("/groups/:groupId/transactions", transactionHandler.CreateGroupTransaction)
		protected.GET("/groups/:groupId/transactions", transactionHandler.GetGroupTransactions)
		protected.POST("/transactions/transfer", transactionHandler.TransferToGroup)
		protected.POST("/groups/:groupId/expenses/pay", transactionHandler.PayGroupExpense)

		// Personal Expenses
		protected.POST("/expenses/personal", expenseHandler.CreatePersonalExpense)
		protected.GET("/expenses/personal", expenseHandler.GetPersonalExpenses)
		protected.GET("/expenses/:expenseId", expenseHandler.GetExpense)
		protected.PUT("/expenses/:expenseId", expenseHandler.UpdateExpense)
		protected.DELETE("/expenses/:expenseId", expenseHandler.DeleteExpense)
		protected.POST("/expenses/:expenseId/buy", expenseHandler.MarkAsBought)
		protected.POST("/expenses/:expenseId/cancel", expenseHandler.MarkAsCancelled)
		protected.GET("/expenses/overdue", expenseHandler.GetOverdueExpenses)

		// Group Expenses
		protected.POST("/groups/:groupId/expenses", expenseHandler.CreateGroupExpense)
		protected.GET("/groups/:groupId/expenses", expenseHandler.GetGroupExpenses)

		// Reports
		protected.GET("/reports/personal/monthly", reportHandler.GetPersonalMonthlyReport)
		protected.POST("/reports/personal/range", reportHandler.GetPersonalDateRangeReport)
		protected.GET("/groups/:groupId/reports/monthly", reportHandler.GetGroupMonthlyReport)
		protected.POST("/groups/:groupId/reports/range", reportHandler.GetGroupDateRangeReport)
		protected.POST("/reports/categories", reportHandler.GetCategoryBreakdown)
		protected.POST("/reports/sources", reportHandler.GetSourceBreakdown)
	}

	// Start server
	addr := cfg.Server.Host + ":" + cfg.Server.Port
	log.Printf("Server starting on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

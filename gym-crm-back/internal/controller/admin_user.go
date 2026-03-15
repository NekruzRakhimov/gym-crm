package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gym-crm/gym-crm-back/internal/models"
	"github.com/gym-crm/gym-crm-back/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type AdminUserController struct {
	adminRepo repository.AdminRepository
}

func NewAdminUserController(adminRepo repository.AdminRepository) *AdminUserController {
	return &AdminUserController{adminRepo: adminRepo}
}

func (ctrl *AdminUserController) List(c *gin.Context) {
	admins, err := ctrl.adminRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list admins"})
		return
	}
	c.JSON(http.StatusOK, admins)
}

func (ctrl *AdminUserController) Create(c *gin.Context) {
	var input models.CreateAdminInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Role != "admin" && input.Role != "manager" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be 'admin' or 'manager'"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	if err := ctrl.adminRepo.CreateWithRole(c.Request.Context(), input.Username, string(hash), input.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create admin"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "admin created"})
}

func (ctrl *AdminUserController) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// Cannot delete yourself
	currentAdminID, _ := c.Get("admin_id")
	if currentAdminID.(int) == id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete yourself"})
		return
	}

	if err := ctrl.adminRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete admin"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "admin deleted"})
}

package api

import (
	"net/http"
	"strings"

	"github.com/nurashi/order-service/internal/service"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	service service.OrderService
}

func NewOrderHandler(service service.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

type CreateOrderRequest struct {
	CustomerID    string `json:"customer_id" binding:"required"`
	CustomerEmail string `json:"customer_email" binding:"required"`
	ItemName      string `json:"item_name" binding:"required"`
	Amount        int64  `json:"amount" binding:"required"`
}

type OrderResponse struct {
	ID            string `json:"id"`
	CustomerID    string `json:"customer_id"`
	CustomerEmail string `json:"customer_email"`
	ItemName      string `json:"item_name"`
	Amount        int64  `json:"amount"`
	Status        string `json:"status"`
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.service.CreateOrder(req.CustomerID, req.CustomerEmail, req.ItemName, req.Amount)
	if err != nil {
		if strings.Contains(err.Error(), "payment service unavailable") ||
			strings.Contains(err.Error(), "timeout") {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := OrderResponse{
		ID:            order.ID,
		CustomerID:    order.CustomerID,
		CustomerEmail: order.CustomerEmail,
		ItemName:      order.ItemName,
		Amount:        order.Amount,
		Status:        string(order.Status),
	}

	c.JSON(http.StatusOK, response)
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")

	order, err := h.service.GetOrder(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}

	response := OrderResponse{
		ID:            order.ID,
		CustomerID:    order.CustomerID,
		CustomerEmail: order.CustomerEmail,
		ItemName:      order.ItemName,
		Amount:        order.Amount,
		Status:        string(order.Status),
	}

	c.JSON(http.StatusOK, response)
}

func (h *OrderHandler) ListOrders(c *gin.Context) {
	orders, err := h.service.GetAllOrders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := make([]OrderResponse, len(orders))
	for i, o := range orders {
		response[i] = OrderResponse{
			ID:            o.ID,
			CustomerID:    o.CustomerID,
			CustomerEmail: o.CustomerEmail,
			ItemName:      o.ItemName,
			Amount:        o.Amount,
			Status:        string(o.Status),
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")

	err := h.service.CancelOrder(id)
	if err != nil {
		if strings.Contains(err.Error(), "only pending orders") {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order cancelled successfully"})
}

func (h *OrderHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/orders", h.CreateOrder)
	router.GET("/orders", h.ListOrders)
	router.GET("/orders/:id", h.GetOrder)
	router.POST("/orders/:id/cancel", h.CancelOrder)
}

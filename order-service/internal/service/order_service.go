package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nurashi/order-service/internal/domain"

	"github.com/google/uuid"
)

type PaymentClient interface {
	ProcessPayment(orderID string, amount int64) (string, error)
}

type httpPaymentClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewHTTPPaymentClient(baseURL string) PaymentClient {
	return &httpPaymentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
	}
}

type processPaymentRequest struct {
	OrderID string `json:"order_id"`
	Amount  int64  `json:"amount"`
}

type processPaymentResponse struct {
	Status string `json:"status"`
}

func (c *httpPaymentClient) ProcessPayment(orderID string, amount int64) (string, error) {
	reqBody := processPaymentRequest{
		OrderID: orderID,
		Amount:  amount,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		c.baseURL+"/payments/process",
		"application/json",
		bytes.NewBuffer(jsonData),
	)

	if err != nil {
		return "", fmt.Errorf("payment service unavailable: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("payment service returned error: %d", resp.StatusCode)
	}

	var paymentResp processPaymentResponse
	if err := json.Unmarshal(body, &paymentResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return paymentResp.Status, nil
}

type OrderService interface {
	CreateOrder(customerID, itemName string, amount int64) (*domain.Order, error)
	GetOrder(id string) (*domain.Order, error)
	CancelOrder(id string) error
}

type orderService struct {
	repo          domain.OrderRepository
	paymentClient PaymentClient
}

func NewOrderService(repo domain.OrderRepository, paymentClient PaymentClient) OrderService {
	return &orderService{
		repo:          repo,
		paymentClient: paymentClient,
	}
}

func (s *orderService) CreateOrder(customerID, itemName string, amount int64) (*domain.Order, error) {
	order := &domain.Order{
		ID:         uuid.New().String(),
		CustomerID: customerID,
		ItemName:   itemName,
		Amount:     amount,
		Status:     domain.OrderStatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.repo.Create(order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	paymentStatus, err := s.paymentClient.ProcessPayment(order.ID, amount)
	if err != nil {
		order.Status = domain.OrderStatusFailed
		order.UpdatedAt = time.Now()
		if updateErr := s.repo.Update(order); updateErr != nil {
			return nil, fmt.Errorf("failed to update order after payment error: %w", updateErr)
		}
		return nil, fmt.Errorf("payment processing failed: %w", err)
	}

	if paymentStatus == "Authorized" {
		order.Status = domain.OrderStatusPaid
	} else {
		order.Status = domain.OrderStatusFailed
	}

	order.UpdatedAt = time.Now()
	if err := s.repo.Update(order); err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	return order, nil
}

func (s *orderService) GetOrder(id string) (*domain.Order, error) {
	return s.repo.GetByID(id)
}

func (s *orderService) CancelOrder(id string) error {
	order, err := s.repo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if order.Status != domain.OrderStatusPending {
		return fmt.Errorf("only pending orders can be cancelled, current status: %s", order.Status)
	}

	order.Status = domain.OrderStatusCancelled
	order.UpdatedAt = time.Now()

	if err := s.repo.Update(order); err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	return nil
}

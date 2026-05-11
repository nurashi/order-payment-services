package provider

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

type SimulatedEmailSender struct{}

func NewSimulatedEmailSender() *SimulatedEmailSender {
	return &SimulatedEmailSender{}
}

func (s *SimulatedEmailSender) Send(to, subject, body string) error {
	time.Sleep(200 * time.Millisecond)

	if rand.Float32() < 0.2 {
		return fmt.Errorf("simulated provider failure: transient network error")
	}

	log.Printf("[SIMULATED EMAIL] to=%s subject=%s body=%s", to, subject, body)
	return nil
}

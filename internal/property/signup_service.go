package property

import (
	"context"
	"errors"
	"sync"
)

var ErrCaptchaRejected = errors.New("captcha rejected")

type MaintenanceRequest struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Status   string `json:"status"`
}

type TenantDocument struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

type InspectionReminder struct {
	ID      string `json:"id"`
	DueDate string `json:"due_date"`
}

type SignupInput struct {
	SignupID              string `json:"signup_id"`
	Email                 string `json:"email"`
	Name                  string `json:"name"`
	CaptchaWidgetRecordID string `json:"captcha_widget_record_id"`
	CaptchaToken          string `json:"captcha_token"`
	IP                    string `json:"-"`
}

type PropertyAccount struct {
	SignupID            string               `json:"signup_id"`
	Email               string               `json:"email"`
	Name                string               `json:"name"`
	MaintenanceRequests []MaintenanceRequest `json:"maintenance_requests"`
	TenantDocuments     []TenantDocument     `json:"tenant_documents"`
	InspectionReminders []InspectionReminder `json:"inspection_reminders"`
}

type CaptchaVerifier interface {
	Verify(context.Context, CaptchaRequest) (CaptchaResult, error)
}

type SignupService struct {
	verifier CaptchaVerifier
	mu       sync.Mutex
	accounts map[string]PropertyAccount
}

func NewSignupService(verifier CaptchaVerifier) *SignupService {
	return &SignupService{verifier: verifier, accounts: make(map[string]PropertyAccount)}
}

func (s *SignupService) Register(ctx context.Context, input SignupInput) (PropertyAccount, bool, error) {
	s.mu.Lock()
	account, exists := s.accounts[input.SignupID]
	s.mu.Unlock()
	if exists {
		return account, false, nil
	}

	result, err := s.verifier.Verify(ctx, CaptchaRequest{
		WidgetRecordID: input.CaptchaWidgetRecordID,
		Token:          input.CaptchaToken,
		Vendor:         "turnstile",
		IP:             input.IP,
		Action:         "property_signup",
		ScoreThreshold: 0.7,
	})
	if err != nil {
		return PropertyAccount{}, false, err
	}
	if !result.Accepted {
		return PropertyAccount{}, false, ErrCaptchaRejected
	}

	account = PropertyAccount{
		SignupID:            input.SignupID,
		Email:               input.Email,
		Name:                input.Name,
		MaintenanceRequests: []MaintenanceRequest{},
		TenantDocuments:     []TenantDocument{},
		InspectionReminders: []InspectionReminder{},
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, ok := s.accounts[input.SignupID]; ok {
		return existing, false, nil
	}
	s.accounts[input.SignupID] = account
	return account, true, nil
}

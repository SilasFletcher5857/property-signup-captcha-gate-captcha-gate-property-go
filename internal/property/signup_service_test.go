package property

import (
	"context"
	"errors"
	"testing"
)

type verifierStub struct {
	result  CaptchaResult
	err     error
	calls   int
	request CaptchaRequest
}

func (v *verifierStub) Verify(_ context.Context, request CaptchaRequest) (CaptchaResult, error) {
	v.calls++
	v.request = request
	return v.result, v.err
}

func TestSignupDecision(t *testing.T) {
	tests := []struct {
		name        string
		result      CaptchaResult
		verifyErr   error
		wantCreated bool
		wantErr     error
	}{
		{name: "verified registration", result: CaptchaResult{Accepted: true}, wantCreated: true},
		{name: "bot registration", result: CaptchaResult{Accepted: false}, wantErr: ErrCaptchaRejected},
		{name: "verification error", verifyErr: errors.New("verification failed"), wantErr: errors.New("verification failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := &verifierStub{result: tt.result, err: tt.verifyErr}
			service := NewSignupService(verifier)
			account, created, err := service.Register(context.Background(), SignupInput{
				SignupID: "signup-1042", Email: "tenant@example.com", Name: "Ada", CaptchaWidgetRecordID: "widget-record-123", CaptchaToken: "browser-token",
			})
			if (err != nil) != (tt.wantErr != nil) {
				t.Fatalf("Register() error = %v, want error %v", err, tt.wantErr)
			}
			if created != tt.wantCreated {
				t.Fatalf("Register() created = %v, want %v", created, tt.wantCreated)
			}
			if created && account.SignupID != "signup-1042" {
				t.Fatalf("Register() signup ID = %q", account.SignupID)
			}
			if verifier.request.WidgetRecordID != "widget-record-123" {
				t.Fatalf("Verify() widget record ID = %q", verifier.request.WidgetRecordID)
			}
		})
	}
}

func TestSignupIDPreventsDuplicateWrite(t *testing.T) {
	verifier := &verifierStub{result: CaptchaResult{Accepted: true}}
	service := NewSignupService(verifier)
	input := SignupInput{SignupID: "signup-1042", Email: "tenant@example.com", CaptchaWidgetRecordID: "widget-record-123", CaptchaToken: "browser-token"}

	if _, created, err := service.Register(context.Background(), input); err != nil || !created {
		t.Fatalf("first Register() created = %v, error = %v", created, err)
	}
	if _, created, err := service.Register(context.Background(), input); err != nil || created {
		t.Fatalf("second Register() created = %v, error = %v", created, err)
	}
	if verifier.calls != 1 {
		t.Fatalf("Verify() calls = %d, want 1", verifier.calls)
	}
}

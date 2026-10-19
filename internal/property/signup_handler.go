package property

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
)

type SignupHandler struct {
	Service *SignupService
}

func (h SignupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method must be POST"})
		return
	}
	var input SignupInput
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || input.SignupID == "" || input.Email == "" || input.CaptchaWidgetRecordID == "" || input.CaptchaToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "signup_id, email, captcha_widget_record_id, and captcha_token are required"})
		return
	}
	input.IP, _, _ = net.SplitHostPort(r.RemoteAddr)

	account, created, err := h.Service.Register(r.Context(), input)
	if err != nil {
		var apiErr *APIError
		switch {
		case errors.Is(err, ErrCaptchaRejected), errors.As(err, &apiErr) && apiErr.HTTPStatus >= 400 && apiErr.HTTPStatus < 500:
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "captcha verification rejected"})
		default:
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "captcha verification could not be completed"})
		}
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, account)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

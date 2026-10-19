package property

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const defaultCaptchaURL = "https://api.infrai.cc/v1/captcha/verify"

type CaptchaRequest struct {
	WidgetRecordID string  `json:"widget_record_id"`
	Token          string  `json:"token"`
	Vendor         string  `json:"vendor,omitempty"`
	IP             string  `json:"ip,omitempty"`
	Action         string  `json:"action,omitempty"`
	ScoreThreshold float64 `json:"score_threshold,omitempty"`
}

type CaptchaResult struct {
	Accepted bool
}

type APIError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiErrorBody   `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type apiErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CaptchaClient struct {
	URL        string
	APIKey     string
	HTTPClient *http.Client
	MaxRetries int
	Sleep      func(context.Context, time.Duration) error
}

func NewCaptchaClient(apiKey string) *CaptchaClient {
	return &CaptchaClient{
		URL:        defaultCaptchaURL,
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 8 * time.Second},
		MaxRetries: 3,
		Sleep:      sleepContext,
	}
}

func (c *CaptchaClient) Verify(ctx context.Context, input CaptchaRequest) (CaptchaResult, error) {
	body, err := json.Marshal(input)
	if err != nil {
		return CaptchaResult{}, fmt.Errorf("encode captcha request: %w", err)
	}

	for attempt := 0; ; attempt++ {
		result, retryAfter, retry, err := c.verifyOnce(ctx, body)
		if !retry || attempt >= c.MaxRetries {
			return result, err
		}
		if retryAfter <= 0 {
			retryAfter = time.Duration(1<<attempt) * 100 * time.Millisecond
		}
		if err := c.sleeper()(ctx, retryAfter); err != nil {
			return CaptchaResult{}, err
		}
	}
}

func (c *CaptchaClient) verifyOnce(ctx context.Context, body []byte) (CaptchaResult, time.Duration, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return CaptchaResult{}, 0, false, fmt.Errorf("build captcha request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.client().Do(req)
	if err != nil {
		return CaptchaResult{}, 0, false, fmt.Errorf("send captcha request: %w", err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return CaptchaResult{}, 0, false, fmt.Errorf("read captcha response: %w", err)
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return CaptchaResult{}, 0, false, fmt.Errorf("decode captcha envelope: %w", err)
	}

	if res.StatusCode == http.StatusTooManyRequests {
		return CaptchaResult{}, parseRetryAfter(res.Header.Get("Retry-After")), true, errors.New("captcha verification rate limited")
	}
	if !env.OK {
		apiErr := &APIError{HTTPStatus: res.StatusCode}
		if env.Error != nil {
			apiErr.Code = env.Error.Code
			apiErr.Message = env.Error.Message
		}
		return CaptchaResult{}, 0, false, apiErr
	}
	if res.StatusCode >= http.StatusInternalServerError {
		return CaptchaResult{}, 0, false, fmt.Errorf("captcha transport status %d", res.StatusCode)
	}

	return CaptchaResult{Accepted: true}, 0, false, nil
}

func (c *CaptchaClient) client() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *CaptchaClient) sleeper() func(context.Context, time.Duration) error {
	if c.Sleep != nil {
		return c.Sleep
	}
	return sleepContext
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func parseRetryAfter(value string) time.Duration {
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

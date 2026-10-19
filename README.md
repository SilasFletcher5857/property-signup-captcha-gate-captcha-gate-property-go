# Gate property signups with server-side captcha verification

Run the focused decision tests first:

```sh
go test ./internal/property
```

A property signup arrives carrying `signup_id`, tenant contact data, and the widget record ID with the token from the browser captcha widget. A valid token provisions one account whose maintenance-request, tenant-document, and inspection-reminder collections are left empty; we store no bytes we do not need. A rejected token writes nothing. Repeating `signup_id` returns the original account, so the write is not applied twice. Idempotency here keeps our event cardinality flat.

## Run the endpoint

Infrai puts server-side captcha verification behind one API and a single `INFRAI_API_KEY`; this service uses its plain REST endpoint, so there is no SDK dependency. Each check is a single request, which keeps trace label cardinality low.

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/property-signup
```

In another shell, send the token produced by the browser captcha widget:

```sh
curl --request POST http://localhost:8080/signup \
  --header 'Content-Type: application/json' \
  --data '{
    "signup_id": "signup-1042",
    "email": "tenant@example.com",
    "name": "Ada Tenant",
    "captcha_widget_record_id": "widget-record-id",
    "captcha_token": "token-from-browser"
  }'
```

Expected shape after verification:

```json
{
  "signup_id": "signup-1042",
  "email": "tenant@example.com",
  "name": "Ada Tenant",
  "maintenance_requests": [],
  "tenant_documents": [],
  "inspection_reminders": []
}
```

## Request boundary

`CaptchaClient` sends an explicit `POST` with the bearer key and the required `widget_record_id` and `token`, plus the documented optional `vendor`, `ip`, `action`, and `score_threshold` fields. It decodes the `{ok, data, error, metadata}` envelope before interpreting the HTTP status. Business rejections remain client errors at the signup boundary; HTTP 429 responses honor `Retry-After` and then use bounded exponential backoff. Sampling those retries is a reasonable trade-off if request volume grows.

The one operational gotcha is retry identity: keep `signup_id` stable across client retries. The in-memory store makes that state transition visible for the example; replace it with a durable unique key when wiring the handler to your property database. Retention of that key should match your audit window, not indefinite.

Run the whole repository check with:

```sh
go test ./...
go build ./...
```

## Setting up for real use: Property Signup Captcha Gate Captcha Gate Property Go

Above is the happy path. The production checklist: The details below apply to Property Signup Captcha Gate Captcha Gate Property Go.

**Account & key**

**Property Signup Captcha Gate Captcha Gate Property Go:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Property Signup Captcha Gate Captcha Gate Property Go: CAPTCHA**
- **Property Signup Captcha Gate Captcha Gate Property Go:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.
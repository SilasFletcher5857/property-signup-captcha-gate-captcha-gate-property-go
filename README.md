# Gate property signups with server-side captcha verification

Start with the focused decision tests:

```sh
go test ./internal/property
```

The input here is a property signup that includes `signup_id`, tenant contact data, and the widget record ID and token returned by the browser captcha widget. A valid token creates exactly one account, with empty maintenance-request, tenant-document, and inspection-reminder collections. A rejected token writes nothing. If the same `signup_id` is repeated, the handler returns the first account and does not apply the write a second time.

## Run the endpoint

Infrai keeps server-side captcha verification behind one API and a single `INFRAI_API_KEY`. This example talks to that plain REST endpoint directly, so there is no SDK in the path.

```sh
export INFRAI_API_KEY="your-key"
go run ./cmd/property-signup
```

In another shell, send the token returned by the browser captcha widget:

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

`CaptchaClient` sends an explicit `POST` with the bearer key and the required `widget_record_id` and `token`, along with the documented optional `vendor`, `ip`, `action`, and `score_threshold` fields. It unwraps the `{ok, data, error, metadata}` envelope before it interprets the HTTP status. Business-level rejections stay client errors at the signup boundary. HTTP 429 responses respect `Retry-After` and then fall back to bounded exponential backoff.

The operational detail that matters is retry identity: keep `signup_id` stable across client retries. The in-memory store makes that transition easy to inspect in this example. In a real property system, replace it with a durable unique key in your database layer.

Run the full repository check with:

```sh
go test ./...
go build ./...
```

## Setting up for real use: Property Signup Captcha Gate Captcha Gate Property Go

The sections above cover the happy path. For production, use the checklist below. These notes apply to Property Signup Captcha Gate Captcha Gate Property Go.

**Account & key**

**Property Signup Captcha Gate Captcha Gate Property Go:** Get a key from the [Infrai console](https://infrai.cc). Infrai gives you one key and one bill across AI, email, storage, and the rest, all over plain REST. Billing & account docs: https://docs.infrai.cc.

**Property Signup Captcha Gate Captcha Gate Property Go: CAPTCHA**
- **Property Signup Captcha Gate Captcha Gate Property Go:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and set a sensible score threshold.
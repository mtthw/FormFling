# FormFling

![FormFling](https://raw.githubusercontent.com/fireph/FormFling/main/web/static/images/banner-2k-transparent.png)

Haven't you always wanted a contact form on your website, but to keep your email private and not spend a ton of money on a paid service? FormFling is a self-hosted contact form service that forwards submissions to your email via SMTP. Think Formspree, but running on your own server.

[![Tests](https://github.com/fireph/formfling/workflows/Run%20Tests/badge.svg)](https://github.com/fireph/formfling/actions/workflows/test.yml)
[![Coverage](https://github.com/fireph/FormFling/wiki/coverage.svg)](https://github.com/fireph/FormFling/wiki/Test-coverage-report)

## Features

- Lightweight and multiplatform
- Gmail SMTP support
- Beautiful HTML email templates
- Origin-based security
- Configured status page return URL
- Health check endpoint
- Docker ready

## Quick Start

### Docker Compose (Recommended)

```bash
git clone https://github.com/FormFling/FormFling.git
cd FormFling
cp docker-compose.yml docker-compose.local.yml
# Edit docker-compose.local.yml with your settings
docker-compose -f docker-compose.local.yml up -d
```

### Docker

```bash
docker run -d \
  --name formfling \
  -p 8080:8080 \
  -e SMTP_USERNAME=your-email@gmail.com \
  -e SMTP_PASSWORD=your-app-password \
  -e FROM_EMAIL=your-email@gmail.com \
  -e TO_EMAIL=recipient@example.com \
  -e ALLOWED_ORIGINS=https://www.yourdomain.com \
  dungfu/form-fling:latest
```

## Configuration

Configure via environment variables:

**Required:**
- `SMTP_USERNAME` - SMTP Username
- `SMTP_PASSWORD` - SMTP Password ([Gmail setup guide](#gmail-setup))
- `FROM_EMAIL` - Sender email
- `TO_EMAIL` - Where to send form submissions

**Optional:**
- `PORT` - Server port (default: 8080)
- `SMTP_HOST` - SMTP server (default: smtp.gmail.com)
- `SMTP_PORT` - SMTP port (default: 587)
- `ALLOWED_ORIGINS` - Comma-separated allowed domains (default: all)
- `FORM_TITLE` - Form name in emails (default: Contact Me)
- `STATUS_REDIRECT_URL` - Optional absolute HTTP(S) URL for the status page's return link and automatic redirect
- `TRUSTED_PROXY_CIDRS` - Optional comma-separated CIDRs for reverse proxies that set `X-Forwarded-For`; default: none
- `RATE_LIMIT_MAX_EMAILS` - Maximum successful emails per visitor IP in each rolling window (default: 3)
- `RATE_LIMIT_WINDOW` - Rolling window as a Go duration, such as `1h`, `24h`, or `48h` (default: `24h`)

See [.env.example](.env.example) for all options.

### Gmail Setup

1. Enable 2-Factor Authentication
2. Generate App Password: Google Account → Security → 2-Step Verification → App passwords
3. Use your email as SMTP username and the generated password as SMTP password

## Usage

Add this form to your website:

```html
<form action="https://your-formfling-domain.com/submit" method="POST">
  <input type="text" name="name" required>
  <input type="email" name="email" required>
  <textarea name="message" minlength="300" required></textarea>
  <button type="submit">Send</button>
</form>
```

### JavaScript/AJAX

To enhance the form above, intercept its submission and ask for JSON. Without JavaScript, it still submits normally and shows the status page.

```javascript
const form = document.querySelector('form');
form.addEventListener('submit', async (event) => {
  event.preventDefault();
  try {
    const response = await fetch(form.action, {
      method: 'POST',
      headers: { Accept: 'application/json' },
      body: new FormData(form)
    });
    const data = await response.json();
    // Show data.status or data.error in your page.
  } catch (error) {
    // Show a network error in your page.
  }
});
```

## API

- `POST /submit` - Submit form
- `GET /health` - Health check
- `GET /status` - Status page

Browser form submissions return to `/status?type=success` or `/status?type=error`. Set `STATUS_REDIRECT_URL` in Docker to show a return link and redirect visitors there after five seconds. The return destination ignores request supplied `redirect`, `Referer`, and `_redirect` values.

Each visitor IP can send up to `RATE_LIMIT_MAX_EMAILS` successful emails in a rolling `RATE_LIMIT_WINDOW`. Further fetch requests receive HTTP 429 and `Retry-After`; regular forms show a message-limit status page. Failed sends do not count. The counter is held in memory, so it resets when the app restarts and is separate for each app instance. If FormFling runs behind a reverse proxy, set `TRUSTED_PROXY_CIDRS` to the proxy's IP range and ensure the proxy appends or replaces `X-Forwarded-For`. Invalid or nonpositive rate-limit settings prevent startup.

**Response format:**
```json
{"status": "message sent"}
{"status": "error", "error": "description"}
```

## Custom Templates

### Custom email template

```bash
docker run -d \
  -v /path/to/template.html:/templates/custom.html \
  -e EMAIL_TEMPLATE=/templates/custom.html \
  dungfu/form-fling:latest
```

Template variables:
- `{{.FormData.Name}}` - Sender's name
- `{{.FormData.Email}}` - Sender's email
- `{{.FormData.Subject}}` - Email subject
- `{{.FormData.Message}}` - Email message
- `{{.FormData.Phone}}` - Sender's phone number
- `{{.FormData.Website}}` - Sender's website
- `{{.SubmittedTime}}` - Time the form was submitted (e.g., "03:04 PM")
- `{{.SubmittedDate}}` - Date the form was submitted (e.g., "02 January 2006")
- `{{.Origin}}` - Origin URL where the form was submitted from

### Custom status template

```bash
docker run -d \
  -v /path/to/template.html:/templates/custom.html \
  -e STATUS_TEMPLATE=/templates/custom.html \
  dungfu/form-fling:latest
```

Template variables:
- `{{.Status}}` - Status type (`success` or `error`)
- `{{.FormTitle}}` - The form title
- `{{.Message}}` - Status message to display
- `{{.RedirectURL}}` - Configured `STATUS_REDIRECT_URL` (if any)

## Development

```bash
git clone https://github.com/fireph/FormFling.git
cd FormFling
go mod download
go run .
```

## Security

- Use HTTPS in production
- Set `ALLOWED_ORIGINS` to restrict access
- Use Gmail App Passwords
- For multiple app instances or limits that survive restarts, also enforce rate limiting at the proxy or with shared storage
- JSON and form submissions are limited to 1 MiB; larger requests receive HTTP 413
- HTTP timeouts: 5 seconds for headers, 15 seconds to read, 60 seconds to write, and 60 seconds idle
- SMTP connections have a 5-second dial timeout and a 20-second deadline for the full send

## License

MIT License - see [LICENSE](LICENSE) file.

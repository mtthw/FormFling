# FormFling

![FormFling](assets/images/banner-2k-transparent.png)

Haven't you always wanted a contact form on your website, but to keep your email private and not spend a ton of money on a paid service? FormFling is a self-hosted contact form service that forwards submissions to your email via SMTP. Think Formspree, but running on your own server.

[![Tests](https://github.com/mtthw/FormFling/actions/workflows/test.yml/badge.svg)](https://github.com/mtthw/FormFling/actions/workflows/test.yml)

## About this fork

This is a fork of [FormFling/FormFling](https://github.com/FormFling/FormFling), tailored for a small contact form hosted on your own server. The website provides the form and its success or error messages; FormFling accepts submissions and sends the email. This fork removes the built-in pages and reCAPTCHA, sends plain text messages with the visitor's address in `Reply-To`, redirects browser submissions back to the website, and adds a honeypot and rate limits per IP address. Releases here are maintained separately from upstream.

## Features

- Lightweight and multiplatform
- No third-party Go dependencies
- SMTP support
- Plain text submission emails with direct reply to the submitter
- Origin-based security
- Direct browser redirects to your contact page
- Health check endpoint
- Docker ready

## Quick Start

### Docker Compose (Recommended)

```bash
git clone https://github.com/mtthw/FormFling.git
cd FormFling
cp .env.example .env
# Edit .env with your SMTP server, credentials, and email addresses
docker compose up -d
```

Compose reads `.env` for local settings. The tracked Compose file supplies defaults for optional values and requires the SMTP credentials and email addresses. It uses the latest versioned release by default. Set `IMAGE_TAG` in `.env` to pin a version, such as `0.5.2`, or use `main` for the newest build from the main branch. `.env` is ignored by Git. To update a running container after a new image is published, run `docker compose pull` followed by `docker compose up -d`.

Pushes to `main` publish `ghcr.io/mtthw/formfling:main`; version tags publish `ghcr.io/mtthw/formfling:latest` and versioned images. New GHCR packages are private by default, so [make the package public](https://docs.github.com/en/packages/learn-github-packages/configuring-a-packages-access-control-and-visibility) for anonymous pulls or sign in to `ghcr.io` before using Compose.

### Docker

```bash
docker run -d \
  --name formfling \
  -p 8080:8080 \
  -e SMTP_USERNAME=your-email@gmail.com \
  -e SMTP_PASSWORD=your-app-password \
  -e FROM_EMAIL=your-email@gmail.com \
  -e TO_EMAIL=recipient@example.com \
  -e STATUS_REDIRECT_URL=https://yourdomain.com/contact \
  -e ALLOWED_ORIGINS=https://www.yourdomain.com \
  ghcr.io/mtthw/formfling:latest
```

## Configuration

Configure via environment variables:

**Required:**
- `SMTP_USERNAME` - SMTP Username
- `SMTP_PASSWORD` - SMTP password or provider token
- `FROM_EMAIL` - Address your SMTP provider permits you to send from (for this fork, set it to `contact@mtthw.xyz`)
- `TO_EMAIL` - Where to send form submissions
- `STATUS_REDIRECT_URL` - Absolute HTTP(S) URL of your contact page for browser form submissions

**Optional:**
- `PORT` - Server port (default: 8080)
- `SMTP_HOST` - SMTP server (default: smtp.gmail.com)
- `SMTP_PORT` - SMTP port (default: 587)
- `ALLOWED_ORIGINS` - Comma-separated allowed domains (default: all)
- `TRUSTED_PROXY_CIDRS` - Optional comma-separated CIDRs for reverse proxies that set `X-Forwarded-For`; default: none
- `RATE_LIMIT_MAX_EMAILS` - Maximum successful emails per visitor IP in each rolling window (default: 3)
- `RATE_LIMIT_WINDOW` - Rolling window as a Go duration, such as `1h`, `24h`, or `48h` (default: `24h`)
- `EXPOSE_VERSION` - Set to `false` to omit the build version from the public health response (default: `true`; invalid values prevent startup)

See [.env.example](.env.example) for all options.

Submission emails use the visitor's name as the display name, `FROM_EMAIL` as the actual sender, and the visitor's email as `Reply-To`. Replies in most mail clients go to the visitor. The submitted subject is used when present; otherwise the subject is `Message from <name>`. The default message is plain text and starts with the visitor's message. Configure your mail provider to authenticate the `FROM_EMAIL` domain (SPF and/or DKIM) for reliable delivery. A form visitor's address should not be used as `From`, because their domain may not authorize your SMTP server to send mail for it.

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
  <textarea name="message" minlength="64" required></textarea>
  <p>Message must be at least 64 characters.</p>
  <div aria-hidden="true" style="position:absolute;left:-10000px">
    <label>Company website <input type="text" name="company_website" tabindex="-1" autocomplete="off"></label>
  </div>
  <button type="submit">Send</button>
</form>
```

Leave `company_website` empty. It is a honeypot: filled submissions receive the normal success response, but no email is sent and no rate-limit slot is used. This catches simple bots; keep the existing rate limit in place as well.

### JavaScript/AJAX

To enhance the form above, intercept its submission and ask for JSON. Without JavaScript, the browser follows a 303 redirect back to the contact page.

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
    // Show a success message if response.ok, otherwise show data.error.
  } catch (error) {
    // Show a network error in your page.
  }
});
```

For visitors without JavaScript, add messages to your contact page that can be shown by the URL fragment:

```html
<style>
  .form-feedback { display: none; }
  .form-feedback:target { display: block; }
</style>
<p id="form-success" class="form-feedback">Your message was sent.</p>
<p id="form-error" class="form-feedback">Your message could not be sent. Please try again.</p>
<p id="form-rate-limited" class="form-feedback">Please wait before sending another message.</p>
```

## API

- `POST /submit` - Submit form
- `GET /health` - Health check and running build version

Check a deployed instance with `curl https://your-formfling-host/health`. By default the response includes a version, for example `{"status":"ok","version":"v0.5.2"}`. Set `EXPOSE_VERSION=false` to return only `{"status":"ok"}`. Published images and binaries built from a tag report that release tag. Builds from later commits on `main` report the nearest tag plus the commit count and short Git hash (for example, `v0.5.2-1-gabc1234`). This distinguishes a moving `main` image from a release. Downloaded binaries also support `formfling --version` without requiring SMTP configuration. Local builds without an injected version report `dev`.

Browser form submissions receive an immediate HTTP 303 to `STATUS_REDIRECT_URL` with `#form-success`, `#form-error`, or `#form-rate-limited`. Any existing fragment in the configured URL is replaced. The redirect ignores request supplied `redirect`, `Referer`, and `_redirect` values.

Each visitor IP can send up to `RATE_LIMIT_MAX_EMAILS` successful emails in a rolling `RATE_LIMIT_WINDOW`. Further fetch requests receive HTTP 429 and `Retry-After`; regular forms redirect to `#form-rate-limited`. Failed sends do not count. The counter is held in memory, so it resets when the app restarts and is separate for each app instance. If FormFling runs behind a reverse proxy, set `TRUSTED_PROXY_CIDRS` to the proxy's IP range and ensure the proxy appends or replaces `X-Forwarded-For`. Invalid or nonpositive rate-limit settings prevent startup.

**Response format:**
```json
{"status": "message sent"}
{"status": "error", "error": "description"}
```

## Development

The [Justfile](Justfile) provides local and Docker Compose commands. With `just` installed, run `just` to list them, `just dev` to start the service from `.env`, or `just check` to run tests, vet, and build.

```bash
git clone https://github.com/mtthw/FormFling.git
cd FormFling
go run .
```

To publish a version, commit your changes on `main`, then run `just release` (patch), `just release minor`, or `just release major`. The recipe checks the clean checkout, fetches tags, runs `just check`, creates an annotated `vMAJOR.MINOR.PATCH` tag, and pushes `main` and the tag together. A pushed tag triggers the release workflows.

## Security

- Use HTTPS in production
- Set `ALLOWED_ORIGINS` to restrict access
- Use Gmail App Passwords
- For multiple app instances or limits that survive restarts, also enforce rate limiting at the proxy or with shared storage
- JSON and form submissions are limited to 1 MiB; fetch requests receive HTTP 413 and browser forms redirect to `#form-error`
- HTTP timeouts: 5 seconds for headers, 15 seconds to read, 60 seconds to write, and 60 seconds idle
- SMTP connections have a 5-second dial timeout and a 20-second deadline for the full send

## License

MIT License - see [LICENSE](LICENSE) file.

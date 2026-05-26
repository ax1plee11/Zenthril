# Sharing Zenthril

Zenthril is an alpha project. Share it as a research and self-hosted testing project, not as a production-ready secure messenger.

## GitHub Repository

Use the repository link:

```text
https://github.com/ax1plee11/Zenthril
```

Recommended wording:

```text
Zenthril is an alpha-stage open-source messaging project focused on realtime systems, hybrid voice, and E2EE research. It is suitable for local testing and controlled self-hosted experiments.
```

## Local Development

```bash
git clone https://github.com/ax1plee11/Zenthril.git
cd Zenthril

cp .env.example .env
# Replace every change-me value before running services.

docker compose up -d
```

Client:

```bash
cd client
npm install
cp .env.example .env
npm run dev
```

## Public Demo Notes

Before sharing a public instance:

- Set `ENVIRONMENT=production`.
- Use exact HTTPS origins in `CORS_ALLOWED_ORIGINS` and `WS_ALLOWED_ORIGINS`.
- Generate strong secrets with `openssl rand -hex`.
- Keep PostgreSQL and Redis private.
- Keep federation disabled unless you are explicitly testing it.
- Do not advertise the current E2EE implementation as Signal-grade.

See:

- `README.md`
- `docs/SECURITY_HARDENING.md`
- `docs/DEPLOYMENT.md`
- `docs/ROADMAP.md`

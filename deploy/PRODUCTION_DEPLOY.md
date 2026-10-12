# Production Deploy Notes

## Trusted Proxies

Grumble derives client addresses for rate limiting and spam detection using the TRUSTED_PROXY_HEADER and TRUSTED_PROXIES environment variables.

- Set TRUSTED_PROXY_HEADER to the header your reverse proxy sets (e.g. X-Forwarded-For, CF-Connecting-IP). Default: X-Forwarded-For.
- Set TRUSTED_PROXIES to a comma-separated list of CIDR ranges that your trusted proxies come from (e.g. 10.0.0.0/8,172.16.0.0/12). When empty, the header is ignored and the connection's remote address is used.
- Never trust the header from untrusted sources. If behind multiple proxies, only trust the ranges you control.
- Changes require restarting the serve process.
# v0.1.2

This release improves fallback behavior when a primary account is unavailable,
disabled, or cooling down inside CLIProxyAPI.

- Adds `cooldown_seconds` so fallback-eligible primary failures can temporarily
  route later requests directly to configured fallback models.
- Treats CPA auth-unavailable, model-cooldown, and operator-disabled-account
  errors as fallback-eligible even when CPA does not provide an HTTP status.
- Keeps SDK compatibility pinned to `github.com/router-for-me/CLIProxyAPI/v7`
  and the native plugin ABI.
# v0.1.2 - Primary fallback cooldown

This release targets the disabled or unavailable primary-account path in
CLIProxyAPI. If a Claude account is manually disabled, unavailable, or placed in
model cooldown, the plugin can now stop hammering the same primary route and send
matching requests directly to configured fallback models for a short cooldown
window.

## What's Changed

- Added `fallback.cooldown_seconds` with a default of `60` seconds.
- Added rule-level `rules[].cooldown_seconds`; set it to `0` globally or per
  rule to disable cooldown.
- During an active cooldown, matching non-streaming and pre-chunk streaming
  requests skip the primary model and try the fallback chain immediately.
- Treated CPA auth-unavailable, model-cooldown, no-active-auth, and
  operator-disabled-account errors as fallback-eligible even when CPA does not
  preserve a numeric HTTP status.
- Avoided duplicate attempts when a fallback model resolves to the same model as
  the primary request.

## SDK Compatibility Notes

The implementation was checked against the pinned CPA SDK module
`github.com/router-for-me/CLIProxyAPI/v7 v7.2.31` and the local CPA fork. CPA's
host model callback exposes model/protocol/body metadata, but not the selected
auth account identity, so cooldown keys are scoped to source format, fallback
rule, and primary model rather than to a concrete account id.

## Assets

The release keeps the CPA plugin store artifact layout: one zip per supported
platform plus `checksums.txt`. Each zip contains exactly one root-level dynamic
library named for the platform.

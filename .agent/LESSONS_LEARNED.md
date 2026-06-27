# Lessons Learned

- CPA built-in retries rotate credentials for the same requested model; they do not switch the requested model name. The plugin fills that gap by calling the CPA host model executor with fallback model names.
- For streaming responses, fallback is only safe before the first payload chunk is emitted.
- CPA can surface quota/rate-limit failures without a numeric HTTP status in the plugin callback path, so text-based quota detection is required as a fallback heuristic.
- Do not commit c-shared build products. Publish them as release assets and include checksums.
- Provider/auth-kind scoped fallback needs host metadata that CPA does not expose to plugin executors today.

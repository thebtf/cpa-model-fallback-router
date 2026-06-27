# Claude Project Notes

Read `AGENTS.md` first. This repository contains a native CPA plugin, not the CPA server itself.

Key constraints:

- Build artifacts are release assets, not source files.
- The plugin currently cannot scope by selected provider/auth kind because CPA does not expose that metadata through the plugin host callback.
- Any future provider-type routing must start by confirming the current CPA plugin ABI and host request metadata.

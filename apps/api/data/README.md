# Runtime Configs

This directory stores local runtime configuration files.

Before starting the backend, copy the examples and fill in your own values:

```bash
cp apps/api/data/ai_config.example.json apps/api/data/ai_config.json
cp apps/api/data/database_config.example.json apps/api/data/database_config.json
```

The real `ai_config.json` and `database_config.json` files are intentionally ignored by git because they can contain API keys, passwords, host addresses, and other machine-specific secrets.

# SE smoke test

The hardware smoke test now lives in `offline-client/offline-client-wails/mpc_core/cmd/se-smoke`.

Run it from the Wails module so it exercises the production `mpc_core/seclient` and `SecurityService` code paths:

```bash
cd offline-client/offline-client-wails
go run ./mpc_core/cmd/se-smoke
```

Useful flags:

```bash
go run ./mpc_core/cmd/se-smoke -reader "GOODIX GSE SmartCard Reader"
go run ./mpc_core/cmd/se-smoke -private-key ../secured/genkey/ec_private_key.pem
go run ./mpc_core/cmd/se-smoke -skip-direct
go run ./mpc_core/cmd/se-smoke -skip-service
```

The old local `seclient` copy was removed to avoid drift from the desktop client's real SE implementation.

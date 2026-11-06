# Getting started – Power Bike

How to start the bike rental prototype. Back to the [README](README.md).

Requirements: [Go](https://go.dev/dl/) 1.27.3 or newer (see `go.mod`). Nothing else – no database, no Node.js, no external Go modules.

1. Start the server from a terminal in the project folder (keep the terminal open while you use the app):

   ```bash
   go run . -demo
   ```

2. Double-click **`open-app.html`** in the project folder – it opens <http://localhost:8080> in your default browser (or type the address yourself). If the browser says the page can't be reached, the server from step 1 is not running.

The server serves the REST API under `/api/` and the frontend from `web/`, so run the command from the project root. Stop it with `Ctrl+C`. All data is kept in memory only and is lost when the server stops.

Options:

```bash
go run .
```

```bash
PORT=9090 go run . -demo
```

| Option | Default | Meaning |
|---|---|---|
| `-demo` | `false` | load demo data at startup: 10 bikes, 4 customers with sample ID copies, reservations that are active, completed, upcoming and cancelled around the current time |
| `PORT` (environment variable) | `8080` | HTTP port; in PowerShell: `$env:PORT=9090; go run . -demo` |

`open-app.html` always opens port 8080 – with a different `PORT`, type the address yourself.

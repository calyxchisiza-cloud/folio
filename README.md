# Folio

Folio indexes documents and returns the ones that share terms with a query. A document that contains more of the query terms comes first.

```text
a 2
b 1
```

`a` contains both `bay` and `whale`. `b` contains `bay`.

## Run

```powershell
go test ./...
go test -race -count=1 ./...
go run ./cmd/demo
```

```powershell
go run ./cmd/folio -listen 127.0.0.1:5320 -dir data
```

```powershell
Invoke-RestMethod -Method Put -Uri http://127.0.0.1:5320/v1/docs/a -Body "blue whale in the bay"
Invoke-RestMethod http://127.0.0.1:5320/v1/search?q=bay+whale
```

The number on each hit is how many distinct query terms that document contains.

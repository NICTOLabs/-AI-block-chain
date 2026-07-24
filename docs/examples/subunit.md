# TENDER Subunit Example

1 TENDER = 10,000,000 HOGOHOGO.

## Formatting amounts

Use the canonical formatter so UI and CLI display the same values:

```go
fmt.Println(FormatAmount(15000000)) // 1 TENDER 5000000 HOGOHOGO
fmt.Println(FormatAmount(5000))     // 0 TENDER 0005000 HOGOHOGO
```

If an API returns raw amounts, divide by `HogohogoPerTender` for tender units.
Keep all on-chain storage in HOGOHOGO for exact arithmetic.

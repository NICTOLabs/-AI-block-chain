# Currency Subunits

The TENDER blockchain uses a fixed subunit to avoid floating-point math:

- 1 TENDER = 10,000,000 HOGOHOGO
- All on-chain balances, fees, and token supply are stored as `uint64` in HOGOHOGO
- The Go helper `FormatAmount` renders amounts for humans

## Code sample

```go
const HogohogoPerTender = 10_000_000

func FormatAmount(amount uint64) string {
    tender := amount / HogohogoPerTender
    hogohogo := amount % HogohogoPerTender
    return fmt.Sprintf("%d TENDER %06d HOGOHOGO", tender, hogohogo)
}
```

### Examples

    FormatAmount(0)              -> "0 TENDER 0000000 HOGOHOGO"
    FormatAmount(1)              -> "0 TENDER 0000001 HOGOHOGO"
    FormatAmount(15_000_000)     -> "1 TENDER 5000000 HOGOHOGO"
    FormatAmount(10_000_000_000) -> "1000 TENDER 0000000 HOGOHOGO"

## Input parsing

```go
func ParseAmount(text string) (uint64, error) {
    text = strings.TrimSpace(text)
    parts := strings.Fields(text)
    tender, _ := strconv.ParseUint(parts[0], 10, 64)
    hogohogo := uint64(0)
    if len(parts) >= 2 {
        v, _ := strconv.ParseUint(parts[1], 10, 64)
        if v < HogohogoPerTender {
            hogohogo = v
        }
    }
    return tender*HogohogoPerTender + hogohogo, nil
}
```

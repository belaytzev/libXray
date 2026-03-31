# Migrate Hysteria Support to Xray-core v26.3.27 QuicParams

## Overview
- Bump Xray-core dependency from v1.260204.0 to v1.260327.0
- Move Hysteria bandwidth (`up`/`down`) and port-hopping (`ports`/`hop-interval`) from deprecated `HysteriaConfig` fields to canonical `FinalMask.QuicParams`
- Add bandwidth/port-hopping support to share link parsing and generation (currently missing)
- Fix TLS parameter gap in hysteria share link parser (reuse `parseSecurity()`)
- Fix pre-existing bug: `Int32Range` uses `Left`/`Right` but `Build()` reads `From`/`To`

## Context (from discovery)
- Files involved: `go.mod`, `share/clash_meta.go`, `share/parse_share.go`, `share/generate_share.go`
- Test file: `share/parse_share_test.go` (new), `share/generate_share_test.go` (new), `share/clash_meta_test.go` (new)
- No existing tests in `share/` package; project uses `testify/assert`
- Helper types in `share/xray_json.go`
- Related Xray-core types: `conf.FinalMask`, `conf.QuicParamsConfig`, `conf.HysteriaConfig`, `conf.Mask`, `conf.Salamander`, `conf.UdpHop`, `conf.Bandwidth`, `conf.Int32Range`

## Key Design Decisions
1. `FinalMask` composed independently from `QuicParams` + Salamander `Udp` masks (3 cases: QuicParams only, Salamander only, both)
2. `Congestion: "brutal"` set only when `Up` or `Down` bandwidth params are present (not for ports-only)
3. Hysteria share link parser reuses `parseSecurity()` for full TLS support (`sni`, `alpn`, `fp`, `ech`, `insecure`)
4. Add `hysteria2`/`hy2` scheme check in `parseSecurity()` for TLS default (like trojan)
5. Hysteria-specific params (`up`/`down`/`ports`/`hop-interval`) in early-return block, not via generic `fm` JSON
6. Fix pre-existing bug: `Int32Range` must use `From`/`To` not `Left`/`Right`
7. `HysteriaSettings` always assigned to `streamSettings` regardless of whether `QuicParams` exists

## Development Approach
- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to the next
- Make small, focused changes
- **CRITICAL: every task MUST include new/updated tests** for code changes in that task
- **CRITICAL: all tests must pass before starting next task**
- **CRITICAL: update this plan file when scope changes during implementation**
- Run tests after each change
- Maintain backward compatibility

## Testing Strategy
- **Unit tests**: required for every task
- Round-trip tests: parse share link -> generate share link -> verify equivalence
- Clash YAML parsing tests: verify bandwidth/port-hopping end up on `QuicParams`
- Edge cases: Salamander + bandwidth combined, bandwidth only, Salamander only, minimal link

## Progress Tracking
- Mark completed items with `[x]` immediately when done
- Add newly discovered tasks with + prefix
- Document issues/blockers with ! prefix
- Update plan if implementation deviates from original scope

## Implementation Steps

### Task 1: Bump Xray-core dependency and fix compilation

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Modify: `share/clash_meta.go` (minimum pointer-type fixes to compile)

Note: The dep bump will break compilation because `HysteriaConfig` fields changed from value types (`Bandwidth`, `UdpHop`) to pointer types (`*Bandwidth`, `*UdpHop`). This task includes minimum fixes to restore compilation; the full migration to `QuicParams` happens in Tasks 2-3.

- [x] Update `go.mod`: `github.com/xtls/xray-core v1.260204.0` -> `v1.260327.0`
- [x] Run `go mod tidy` to update transitive dependencies
- [x] Fix pointer-type compilation breaks in `clash_meta.go` (temporary — use pointer assignments for `Up`/`Down`/`UdpHop`)
- [x] Run `go build ./...` to verify compilation succeeds
- [x] Write smoke test verifying `conf.QuicParamsConfig` struct is accessible (confirms dep updated correctly)
- [x] Run existing tests: `go test ./...`

### Task 2: Refactor share link parsing — hysteria with QuicParams and parseSecurity

**Files:**
- Modify: `share/parse_share.go`
- Create: `share/parse_share_test.go`

- [x] In `parseSecurity()`: add `hysteria2` and `hy2` scheme check alongside `trojan` for TLS default (line ~630)
- [x] Rewrite `hysteriaOutbound()` to:
  - Keep `HysteriaConfig` with `Version` + `Auth` only
  - Parse `up`, `down` query params -> build `QuicParamsConfig` with `Congestion: "brutal"` (only when up/down present)
  - Parse `ports`, `hop-interval` query params -> set `QuicParams.UdpHop` with `From`/`To` (not `Left`/`Right`)
  - Build Salamander `udpMasks` independently from `obfs-password`
  - Compose single `FinalMask` from QuicParams + Udp when either present
  - Always assign `HysteriaSettings` to `streamSettings`
  - Delegate TLS to `parseSecurity()` instead of inline SNI-only handling
- [x] Write tests: minimal hy2 link `hy2://auth@host:443?sni=example.com`
- [x] Write tests: hy2 with bandwidth `hy2://auth@host:443?up=100+mbps&down=200+mbps&sni=example.com`
- [x] Write tests: hy2 with Salamander `hy2://auth@host:443?obfs=salamander&obfs-password=secret&sni=example.com`
- [x] Write tests: hy2 with everything `hy2://auth@host:443?up=50+mbps&down=100+mbps&obfs=salamander&obfs-password=secret&ports=20000-40000&hop-interval=30&sni=example.com`
- [x] Write tests: hy2 with TLS params `hy2://auth@host:443?sni=example.com&alpn=h3&fp=chrome`
- [x] Write tests: hy2 with ports only (no bandwidth) — verify no `Congestion` set
- [x] Run tests: `go test ./share/...`

### Task 3: Refactor Clash Meta parser — hysteria with QuicParams

**Files:**
- Modify: `share/clash_meta.go`
- Create: `share/clash_meta_test.go`

- [x] Rewrite `streamSettings()` hysteria case to:
  - Keep `HysteriaSettings` for `Version` + `Auth` only
  - Build `QuicParamsConfig` independently when `proxy.Up`/`proxy.Down`/`proxy.Ports` present
  - Set `Congestion: "brutal"` only when `Up` or `Down` present
  - Only set `BrutalUp`/`BrutalDown` when respective values are non-empty
  - Set `QuicParams.UdpHop` with `From`/`To` (fix `Left`/`Right` bug)
  - Build Salamander `udpMasks` independently
  - Compose single `FinalMask` from both QuicParams + Udp
  - Always assign `HysteriaSettings` to `streamSettings`
- [x] In Clash `parseSecurity()`: add `outbound.Protocol == "hysteria"` alongside trojan check for TLS default (line ~491)
- [x] Write tests: Clash hysteria2 with bandwidth + port-hopping + Salamander
- [x] Write tests: Clash hysteria2 with bandwidth only (no obfs, no ports)
- [x] Write tests: Clash hysteria2 with Salamander only
- [x] Write tests: Clash hysteria2 minimal (auth only)
- [x] Run tests: `go test ./share/...`

### Task 4: Update share link generation — output QuicParams for hysteria

**Files:**
- Modify: `share/generate_share.go`
- Create: `share/generate_share_test.go`

- [x] In `streamSettingsQuery()` hysteria early-return block, add QuicParams output (guard with `FinalMask != nil && FinalMask.QuicParams != nil` to avoid panic when FinalMask only has Salamander):
  - `up` from `FinalMask.QuicParams.BrutalUp` (cast to `string()` — `Bandwidth` is a string type alias)
  - `down` from `FinalMask.QuicParams.BrutalDown`
  - `ports` from `FinalMask.QuicParams.UdpHop.PortList`
  - `hop-interval` from `FinalMask.QuicParams.UdpHop.Interval`
- [x] Add full TLS param output in hysteria early-return block (`fp`, `alpn`, `ech`, `pcs`, `vcn`) — not just `sni` — to match the richer TLS parsing from Task 2. Reuse the existing TLS/REALITY output logic or extract shared helper.
- [x] Keep existing Salamander output unchanged
- [x] Write tests: generate hy2 link with bandwidth params
- [x] Write tests: generate hy2 link with Salamander + bandwidth + port-hopping
- [x] Write tests: generate hy2 link with full TLS params (`sni`, `alpn`, `fp`)
- [x] Write tests: round-trip — parse hy2 link then generate, verify key params preserved (including TLS params)
- [x] Run tests: `go test ./share/...`

### Task 5: Verify acceptance criteria

- [x] Verify all requirements from Overview are implemented
- [x] Verify edge cases: QuicParams only, Salamander only, both combined
- [x] Verify `Int32Range` bug is fixed (uses `From`/`To`)
- [x] Verify `parseSecurity()` TLS default works for hysteria2/hy2 schemes
- [x] Run full test suite: `go test ./...`
- [x] Run `go vet ./...` for static analysis

### Task 6: [Final] Update documentation

- [ ] Update README.md if needed
- [ ] Run final `go test ./...` and `go vet ./...` to confirm no regressions
- [ ] Move this plan to `docs/plans/completed/`

## Technical Details

### QuicParamsConfig struct (from Xray-core v26.3.27)
```go
type QuicParamsConfig struct {
    Congestion  string    `json:"congestion"`   // "brutal", "bbr", "reno", "force-brutal"
    BrutalUp    Bandwidth `json:"brutalUp"`      // e.g. "100 mbps"
    BrutalDown  Bandwidth `json:"brutalDown"`    // e.g. "200 mbps"
    UdpHop      UdpHop    `json:"udpHop"`        // ports + interval
    // ... QUIC window params omitted (not relevant for share links)
}
```

### FinalMask struct (v26.3.27)
```go
type FinalMask struct {
    Tcp        []Mask            `json:"tcp"`
    Udp        []Mask            `json:"udp"`         // Salamander goes here
    QuicParams *QuicParamsConfig `json:"quicParams"`  // NEW: bandwidth/hop goes here
}
```

### HysteriaConfig (v26.3.27 — deprecated fields)
```go
type HysteriaConfig struct {
    Version int32  `json:"version"`  // keep using
    Auth    string `json:"auth"`     // keep using
    // DEPRECATED (now pointers, log warnings):
    Congestion *string    `json:"congestion"`
    Up         *Bandwidth `json:"up"`
    Down       *Bandwidth `json:"down"`
    UdpHop     *UdpHop    `json:"udphop"`
}
```

### Int32Range fix
```go
// WRONG (current code):
interval.Left = proxy.HopInterval
interval.Right = proxy.HopInterval

// CORRECT:
interval.From = proxy.HopInterval
interval.To = proxy.HopInterval
```

### FinalMask composition pattern (used in Tasks 2, 3)
```go
// Build QuicParams (independent)
var quicParams *conf.QuicParamsConfig
if len(up) > 0 || len(down) > 0 || len(ports) > 0 {
    quicParams = &conf.QuicParamsConfig{}
    if len(up) > 0 || len(down) > 0 {
        quicParams.Congestion = "brutal"
    }
    if len(up) > 0 {
        quicParams.BrutalUp = conf.Bandwidth(up)
    }
    if len(down) > 0 {
        quicParams.BrutalDown = conf.Bandwidth(down)
    }
    if len(ports) > 0 {
        quicParams.UdpHop = conf.UdpHop{PortList: ..., Interval: &conf.Int32Range{From: interval, To: interval}}
    }
}

// Build Salamander (independent)
var udpMasks []conf.Mask
if len(obfsPassword) > 0 {
    // existing salamander mask logic
}

// Compose FinalMask
if quicParams != nil || len(udpMasks) > 0 {
    streamSettings.FinalMask = &conf.FinalMask{QuicParams: quicParams, Udp: udpMasks}
}
```

## Post-Completion

**Manual verification:**
- Test with real hysteria2 share links from production servers
- Verify round-trip with third-party clients (v2rayN, NekoBox) for interoperability

**External system updates:**
- Consuming mobile apps (Android/iOS) that use libXray may need rebuilding
- Verify gomobile bindings still compile after dep bump

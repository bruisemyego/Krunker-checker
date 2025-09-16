# Krunker Account Checker

**Notice:** If the checker feels slow, it's most likely because your proxies are trash. Use good proxies for best results.

https://www.youtube.com/watch?v=favX310xSS4

## DM me on Discord @cleanest for more info or to purchase.
**Krunker Botter — \$50 (source included)**: Spawn bots that join your game so you can flood matches or farm nukes/KR.

**Krunker Siphone (KR Farm) — \$70 (source included)**: login to accounts → join a game → deposit KR into the map. Includes a verifier api that auto-verifies FRVR verification

## Credits/Contact
- Discord: [Join Server](https://discord.gg/QgqKpKVG5t)
- Dev: @cleanest on discord

## Features
- Complete account stats fetching (Level, Inventory Value, KR)
- Smart proxy management with auto rotation and cleanup
- Multi-threaded account checking (default 500 threads)
- Handles both username and email login endpoints
- Automatic captcha solving (SHA-256)
- Real-time stats and CPM counter
- User agent rotation (Chrome, Firefox, Edge)
- Built-in proxy scraper

## Setup
1. Install Go (1.21+)
2. Clone this repo
3. Run `go mod tidy`

## File Structure
```
├── main.go
├── src/
│   ├── login.go
│   ├── profile.go
│   ├── proxy.go
│   ├── scraper.go
│   ├── captcha.go
│   └── utils.go
├── data/
│   ├── accounts.txt
│   └── proxies.txt
└── results/
    └── (result files)
```

## Usage
1. Put accounts in `data/accounts.txt` (user:pass or email:pass format)
2. Run: `go run main.go`
3. Choose whether to scrape proxies automatically (y/N)
4. Set thread count (or press enter for default 500)

**Note:** The checker will automatically scrape proxies if none are found, or you can add your own to `data/proxies.txt`

## Proxy Format
The checker supports multiple proxy formats:
- `ip:port` (will auto-add http://)
- `http://ip:port`
- `https://ip:port` 
- `socks5://ip:port`
- `username:password@ip:port`

## Results
- `login_ok.txt` - Working accounts
- `needs_migrate.txt` - Need email migration
- `needs_verification.txt` - Need email verification
- `bad_accounts.txt` - Invalid credentials  
- `undetermined.txt` - Couldn't check due to proxy issues

## Performance Tips
- Use quality proxies for best speed and accuracy
- Higher thread counts work better with more proxies
- The checker automatically cleans up bad proxies for you

# ⭐ Please star the repo if you found this useful!










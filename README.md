# Krunker Account Checker V2
# ⭐ Please star for V3 with even better improvements ⭐

Account checker for Krunker.io with proxy support and captcha solving.  

⚠️ **Notice:** If the checker feels slow, it’s most likely because your proxies are trash. Use good proxies for best results.  

## V2 Features
- Multi-threaded account checking (default 500 threads)
- Proxy support (HTTP/HTTPS/SOCKS5) with auto rotation
- Automatic captcha solving (SHA-256)
- User agent rotation (Chrome, Firefox, Edge)
- Real-time stats and CPM counter
- Detailed results categorization
- Bad proxy cleanup

## Setup

1. Install Go (1.21+)
2. Clone this repo
3. Run `go mod tidy`

## Usage

1. Put accounts in `data/accounts.txt` (username:password or email:password)
2. Add proxies to `data/proxies.txt` 
   - Format: `ip:port` or `protocol://ip:port`
   - Supports http://, https://, socks5://
3. Run: `go run main.go`
4. Choose thread count (or press enter for 500)

## Results

Results are saved in the `results/` folder:
- `login_ok.txt` - Working accounts
- `needs_migrate.txt` - Need email migration
- `needs_verification.txt` - Need email verification
- `Banned.txt` - Account Banned
- `bad_accounts.txt` - Bad credentials
- `undetermined.txt` - Couldn't check because of bad proxies

## File Structure
```

├── main.go
├── services/
│   ├── login.go
│   └── captcha.go
├── data/
│   ├── accounts.txt
│   └── proxies.txt
└── results/
└── (result files)

```
## Contact

- Discord: [Join Server](https://discord.gg/QgqKpKVG5t)
- Dev: @cleanest on discord

## Credits

Coded by @cleanest

---
⭐ Get this to 10 stars for V3 with even better improvements ⭐




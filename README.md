# Krunker Account Checker

Account checker for Krunker.io with proxy support and captcha solving capabilities

## Features
- Multi-threaded account checking
- Proxy support (HTTP/HTTPS/SOCKS5)
- Automatic captcha solving
- User agent rotation
- Detailed results categorization

## Installation

1. Install Go (1.21 or later)
2. Clone the repository
3. Install dependencies:
```bash
go mod tidy
```

## Usage

1. Place your accounts in `data/accounts.txt` (format: username:password)
2. Add proxies in `data/proxies.txt` (required)
   - Format: ip:port or protocol://ip:port
   - Supported protocols: http://, https://, socks5://
   - If no protocol is specified, http:// will be used
3. Run the checker:
```bash
go run main.go
```

The results will be saved in the `results` folder:
- `login_ok.txt`: Working accounts
- `needs_migrate.txt`: Accounts that need migration

## Contact & Support

- Discord Server: [Join Here](https://discord.gg/QgqKpKVG5t)
- Developer: [@cleanest](https://discord.com/users/cleanest)
- For help and support, contact @cleanest on Discord

## Credits

Coded by @cleanest

---
⭐ Don't forget to star the repo if you find it useful!
⭐ Don't forget to star the repo if you find it useful!


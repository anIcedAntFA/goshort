# GoShort Demo — shows API + CLI in action
# Run: vhs docs/demo.tape

Output docs/demo.gif

Set FontSize 16
Set Width 1200
Set Height 600
Set TypingSpeed 40ms
Set Padding 20
Set BorderRadius 12
Set WindowBar Colorful
Set Theme "Catppuccin Mocha"

# Build binaries silently so the demo starts clean
Hide
Type "make build"
Enter
Sleep 8s
Show

# Start server — redirect logs so they don't interleave with demo output
Type "GOSHORT_CACHE_DRIVER=memory ./bin/goshort > /tmp/goshort.log 2>&1 &"
Enter
Sleep 2s

# Create a short URL
Type `curl -s -X POST http://localhost:8080/api/v1/urls -H 'Content-Type: application/json' -d '{"url":"https://github.com/anIcedAntFA/goshort"}' | jq .`
Enter
Sleep 3s

# Create with custom alias
Type `curl -s -X POST http://localhost:8080/api/v1/urls -H 'Content-Type: application/json' -d '{"url":"https://go.dev","custom_alias":"go-lang"}' | jq .`
Enter
Sleep 3s

# CLI — list URLs
Type "./bin/goshort-cli list"
Enter
Sleep 2s

# CLI — stats for a specific alias
Type "./bin/goshort-cli stats go-lang"
Enter
Sleep 2s

# CLI — shorten a long URL directly
Type "./bin/goshort-cli shorten https://example.com/very/long/path"
Enter
Sleep 2s

# Cleanup
Type "kill %1"
Enter
Sleep 500ms

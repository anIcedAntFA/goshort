# GoShort Demo — shows API + CLI in action
# Run: vhs docs/demo.tape

Output docs/demo.gif

Set FontSize 14
Set Width 900
Set Height 500
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms
Set Padding 20

# Start server
Type "GOSHORT_CACHE_DRIVER=memory ./bin/goshort &"
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
Type "goshort-cli list"
Enter
Sleep 2s

# CLI — stats
Type "goshort-cli stats go-lang"
Enter
Sleep 2s

# CLI — pipe support
Type "echo 'https://example.com/very/long/path' | goshort-cli shorten"
Enter
Sleep 2s

# Cleanup
Type "kill %1"
Enter
Sleep 500ms

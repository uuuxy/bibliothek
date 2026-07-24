import re

with open("mailservice/mailservice.go", "r") as f:
    content = f.read()

# Let's see if there is any other way to fix this without annotations.
# In go, `strings.ReplaceAll` is a sanitizer if it replaces '\r' and '\n'.
# We replaced them in `betreff`, `to`, `c.Sender`.
# But `bodyText` is not sanitized! Because `bodyText` shouldn't be sanitized.
# BUT we can sanitize `msg` right before sending, just replacing `\r` with ` ` maybe? No, SMTP needs `\r\n`.
# Can we use `mime.WordEncoder`?

# If `lgtm[go/mail-injection]` doesn't work, maybe we should just use `net/mail` Message?
# But `net/mail` does not have a `SendMessage` method.

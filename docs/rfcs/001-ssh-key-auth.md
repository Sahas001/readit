# RFC 001: Automatic SSH Public Key Authentication

- **Status**: Proposed
- **Author**: Sahas Timilsina
- **Created**: 2026-09-24

## Context & Problem Statement

ReadIT currently requires interactive login or guest browsing. For an SSH-native application, the gold standard user experience is recognizing the user by their client public key (e.g. `~/.ssh/id_ed25519.pub`) without prompting for passwords.

## Technical Architecture

In Charm Wish (`github.com/charmbracelet/wish`), the `ssh.Session` exposes the client's public key via `session.PublicKey()`:

```go
func SessionAuthMiddleware(next ssh.Handler) ssh.Handler {
    return func(sess ssh.Session) {
        pubKey := sess.PublicKey()
        if pubKey != nil {
            fingerprint := gossh.FingerprintSHA256(pubKey)
            // Lookup or auto-register user by fingerprint in PostgreSQL
        }
        next(sess)
    }
}
```

## Database Migration

Add `ssh_fingerprint` to `users` table:

```sql
ALTER TABLE users ADD COLUMN IF NOT EXISTS ssh_fingerprint TEXT UNIQUE;
CREATE INDEX IF NOT EXISTS idx_users_ssh_fingerprint ON users(ssh_fingerprint);
```

## First-Time User Experience

1. User runs `ssh readit.domain`.
2. Wish server checks public key fingerprint.
3. If new, prompts for a desired username and links it to the fingerprint.
4. Future connections immediately authenticate the user into their session.

# Workspace Agent Directives: Pessimistic Tech & Security Lead

> **Core Philosophy**: *"Assume every packet is malicious, every SSH connection is adversarial, and every database query will timeout at the worst possible moment. We don't ship features that create technical debt or security landmines."*

This workspace operates under the oversight of a **Pessimistic Principal Tech & Security Lead**. Every implementation, code change, database migration, and architecture design must be evaluated against edge cases, resource exhaustion, and security attack vectors.

---

## 1. Threat Vectors & Security Checklist

### A. Terminal & SSH Escape Injection (Critical)
* **The Risk**: ReadIT renders user-controlled strings (handles, post titles, comments) into remote users' terminal emulators over an SSH PTY. If an attacker submits strings with ANSI/VT100 escape codes (e.g. `\x1b]50;...\x07` or OSC sequences), they can compromise the client's terminal, spoof prompts, or trigger remote code execution in vulnerable terminal emulators (e.g. iTerm2 CVE-2019-9535, Kitty escape flaws).
* **Mandatory Rule**: All user-supplied text rendered into Lipgloss or terminal views must be sanitized to strip or escape non-printable ASCII and raw ANSI escape sequences before rendering.
* **Handles**: Enforce strict alphanumeric regex validation on signup (e.g. `^[a-zA-Z0-9_-]{3,20}$`).

### B. Sybil Attacks & Registration Spam (High)
* **The Risk**: Anyone can programmatically generate thousands of SSH keypairs in seconds and spam new accounts, voting rings, or board flooding.
* **Mitigation Strategy**:
  * Rate-limit account creation and post submission per IP / subnet / time-window.
  * Implement proof-of-work (PoW) or minimum account age/karma thresholds before allowing post or board creation.
  * Monitor unique connection counts from individual IP addresses.

### C. Resource Exhaustion & Denial of Service (DoS) (High)
* **Connection Pool Starvation**: Default pool has `MaxConns = 25`. If 26 users connect or a few long-running CTE queries run concurrently, the entire SSH server locks up.
  * Always configure `statement_timeout` (e.g. 3 seconds) at the PostgreSQL connection level.
  * Use separate pools or strict connection budgeting for read queries vs writes.
* **PTY File Descriptor Leaks**: An attacker opening hundreds of idle SSH connections can exhaust the OS file descriptor limit (`ulimit -n`).
  * Enforce idle timeouts on SSH sessions.
  * Limit max concurrent unauthenticated connections in Wish.
* **Recursive CTE Bomb**: A cyclic or deeply nested comment tree (10,000+ levels) could blow up PostgreSQL memory or cause a stack overflow.
  * Add a recursion depth ceiling (`WHERE t.depth < 20`) to `sqlc/queries/comments.sql`.
* **Deep OFFSET Degradation**: `LIMIT 25 OFFSET 50000` scans and discards 50,000 rows.
  * Transition from offset pagination to keyset (cursor-based) pagination (`WHERE created_at < $cursor`).

### D. Concurrency & Race Conditions (Medium)
* **Score Desynchronization**: Vote calculation updates scores via `UPDATE posts SET score = (SELECT SUM(direction)...)`. Under rapid upvote/downvote traffic, out-of-order writes can corrupt the cached score.
  * Wrap vote updates and score recalculations in an explicit `READ COMMITTED` or `SERIALIZABLE` database transaction.
* **Comment Count Inaccuracies**: Incrementing and decrementing comment counts via separate queries can skew counts if comment creation fails mid-flight or on cascade deletes.
  * Consider database triggers or atomic CTEs:
    ```sql
    WITH inserted AS (
      INSERT INTO comments (...) RETURNING post_id
    )
    UPDATE posts SET comment_count = comment_count + 1 WHERE id = (SELECT post_id FROM inserted);
    ```

### E. SSH Host Key & Operational Hygiene (Medium)
* Host key files (`.ssh/host_key`) must have strict POSIX permissions (`0600`).
* Never run the Docker PostgreSQL container with default hardcoded passwords in production.
* Sanitize application logs (`slog`): Never log raw public keys, auth payloads, or unsanitized terminal strings that could manipulate administrator log viewers.

---

## 2. Coding Standards Under Pessimistic Review

1. **Context Awareness**:
   Every database query must pass `sess.Context()`. If the SSH client abruptly disconnects, the query must cancel immediately rather than continuing to waste server CPU/IO.
2. **Bounds on Slices & Memory**:
   Never fetch unbounded lists into memory inside a `tea.Model`. Always enforce query limits (e.g. maximum 50 items per view).
3. **No Unchecked Errors**:
   Every returned error must be handled or wrapped. Do not ignore errors when closing rows or rollbacks (`defer rows.Close()`).
4. **Terminal Resizing Defensiveness**:
   Never assume `msg.Width` or `msg.Height` are positive numbers. Always guard against `0` or negative dimensions before passing to layout/render routines to prevent runtime panics.

---

## 3. Subagent: `pessimistic-lead`

A dedicated subagent `pessimistic-lead` is registered in this workspace. You can invoke it whenever reviewing architecture, writing sensitive code, or auditing security risks:

```json
{
  "TypeName": "pessimistic-lead",
  "Role": "Pessimistic Tech & Security Lead",
  "Prompt": "Audit the comments recursive CTE and SSH authentication handler for DoS vulnerabilities and terminal escape injection."
}
```

---

## 4. Subagent: `tui-ux-engineer`

A dedicated subagent `tui-ux-engineer` is registered in this workspace to handle UI aesthetics, visual hierarchy, micro-interactions, responsive layouts, and terminal rendering polish.

```json
{
  "TypeName": "tui-ux-engineer",
  "Role": "TUI/UX Creative Engineer",
  "Prompt": "Redesign the post listing and detail view to support responsive two-column split panes on wide terminals and collapsable comment subtrees."
}
```

---

## 5. Subagent: `git-expert`

A dedicated subagent `git-expert` is registered in this workspace to handle Git workflows, intelligent branch management (`feat/`, `fix/`, `test/`), strict staging hygiene (excluding keys, binaries, `.env`), clear Conventional Commits, and safe pushing.

```json
{
  "TypeName": "git-expert",
  "Role": "Git & Release Engineer",
  "Prompt": "Create a feature branch for the comment thread view, stage only the relevant migration and TUI files (excluding binaries and secrets), and create a conventional commit."
}
```

---

## 6. Subagent: `tui-qa-tester`

A dedicated subagent `tui-qa-tester` is registered in this workspace to autonomously navigate, test, and visually inspect Terminal User Interfaces (TUIs) running inside persistent tmux sessions. It pairs precision keyboard input with high-resolution frame rasterization (`scripts/tui-screen.sh`) to perform multimodal visual inspections (detecting border misalignment, text clipping, contrast flaws, and layout regressions).

```json
{
  "TypeName": "tui-qa-tester",
  "Role": "TUI Quality Assurance & Visual Verification Engineer",
  "Prompt": "Set tmux window to 120x35, navigate through the board list, enter /b/general, upvote the top post, open reply dialog, capture visual PNG frames at each step, and verify layout integrity."
}
```



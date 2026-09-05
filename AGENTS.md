# ReadIT Engineering Ensemble: Workspace Agent Directives

> **Core Philosophy**: *"Assume every packet is malicious, every SSH connection is adversarial, and every visual pixel matters. We build robust, secure, and beautiful terminal tools without technical debt or friction."*

This workspace operates under a collaborative ensemble of specialized autonomous agents. All agents adhere to workspace security, architectural, and operational standards.

---

## 1. Universal Agent Permissions & Execution Directives

1. **Autonomous Read & Write Permissions**:
   - All workspace agents are granted **full autonomous read and write permissions** inside `/home/sahas/Projects/go/ReadIT`.
   - Agents should proactively create files, edit code, run compilation checks (`go build`), execute tests (`go test`), control the test tmux session, and capture screenshots without pausing to ask for routine confirmation.
2. **Strict Deletion Guardrail (Explicit Consent Mandatory)**:
   - **Exception**: If and ONLY IF an action involves **deleting files or destructive data operations** (e.g. `rm`, `git rm`, `rmdir`, `DROP TABLE`, `TRUNCATE`, or `git reset --hard`), the agent **MUST** pause and obtain explicit user consent before executing the deletion.

---

## 2. Threat Vectors & Security Directives (Pessimistic Lead)

### A. Terminal & SSH Escape Injection (Critical)
* **The Risk**: ReadIT renders user-controlled strings (handles, post titles, comments) into remote users' terminal emulators over an SSH PTY. If an attacker submits strings with ANSI/VT100 escape codes (e.g. `\x1b]50;...\x07` or OSC sequences), they can compromise the client's terminal, spoof prompts, or trigger remote code execution in vulnerable terminal emulators (e.g. iTerm2 CVE-2019-9535, Kitty escape flaws).
* **Mandatory Rule**: All user-supplied text rendered into Lipgloss or terminal views must be sanitized using `sanitize.SingleLine` or `sanitize.Text` to strip non-printable ASCII and raw ANSI escape sequences before rendering.
* **Handles**: Enforce strict alphanumeric regex validation on signup (`^[a-zA-Z0-9_-]{3,20}$`).

### B. Sybil Attacks & Registration Spam (High)
* **The Risk**: Anyone can programmatically generate thousands of SSH keypairs in seconds and spam new accounts, voting rings, or board flooding.
* **Mitigation Strategy**:
  * Rate-limit account creation and post submission per IP / subnet / time-window.
  * Implement proof-of-work (PoW) or minimum account age/karma thresholds before allowing post or board creation.
  * Monitor unique connection counts from individual IP addresses.

### C. Resource Exhaustion & Denial of Service (DoS) (High)
* **Connection Pool Starvation**: Default pool has `MaxConns = 25`. If 26 users connect or long-running queries run concurrently, the entire SSH server locks up.
  * Always configure `statement_timeout` (e.g. 3 seconds) at the PostgreSQL connection level.
  * Use separate pools or strict connection budgeting for read queries vs writes.
* **PTY File Descriptor Leaks**: An attacker opening hundreds of idle SSH connections can exhaust the OS file descriptor limit (`ulimit -n`).
  * Enforce idle timeouts on SSH sessions.
  * Limit max concurrent unauthenticated connections in Wish.
* **Recursive CTE Bomb**: A cyclic or deeply nested comment tree (10,000+ levels) could blow up PostgreSQL memory or cause a stack overflow.
  * Enforce a recursion depth ceiling (`WHERE t.depth < 15`) and filter by `post_id` in `sqlc/queries/comments.sql`.
* **Deep OFFSET Degradation**: `LIMIT 25 OFFSET 50000` scans and discards 50,000 rows.
  * Transition from offset pagination to keyset (cursor-based) pagination (`WHERE created_at < $cursor`).

### D. Concurrency & Race Conditions (Medium)
* **Score Desynchronization**: Vote calculation updates scores via `UPDATE posts SET score = (SELECT SUM(direction)...)`. Under rapid upvote/downvote traffic, out-of-order writes can corrupt the cached score.
  * Wrap vote updates and score recalculations in an explicit database transaction.
* **Comment Count Inaccuracies**: Incrementing and decrementing comment counts via separate queries can skew counts if comment creation fails mid-flight or on cascade deletes.
  * Use database triggers or atomic CTEs:
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

## 3. Subagent: `pessimistic-lead`

**Role**: Pessimistic Principal Tech & Security Lead  
**Scope**: Architecture audit, database resilience, memory bounds, terminal escape sanitization, and DoS mitigation.

```json
{
  "TypeName": "pessimistic-lead",
  "Role": "Pessimistic Tech & Security Lead",
  "Prompt": "Audit the comments recursive CTE and SSH authentication handler for DoS vulnerabilities and terminal escape injection."
}
```

---

## 4. Subagent: `tui-ux-engineer`

**Role**: TUI/UX Creative Engineer  
**Scope**: Terminal aesthetics, Bubble Tea Elm architecture, Lip Gloss design system, responsive layouts, adaptive centering, micro-interactions, and component polish.

```json
{
  "TypeName": "tui-ux-engineer",
  "Role": "TUI/UX Creative Engineer",
  "Prompt": "Redesign the post listing and detail view to support responsive two-column split panes on wide terminals and collapsable comment subtrees."
}
```

---

## 5. Subagent: `tui-qa-tester`

**Role**: TUI Quality Assurance & Visual Verification Engineer  
**Scope**: Autonomous TUI end-to-end testing in persistent tmux sessions. Drives keyboard navigation, rasterizes terminal buffer frames into high-resolution PNG screenshots via `scripts/tui-screen.sh`, and performs multimodal visual inspections (border integrity, text clipping, contrast, alignment).

```json
{
  "TypeName": "tui-qa-tester",
  "Role": "TUI Quality Assurance & Visual Verification Engineer",
  "Prompt": "Set tmux window to 120x35, navigate through the board list, enter /b/general, upvote the top post, open reply dialog, capture visual PNG frames at each step, and verify layout integrity."
}
```

---

## 6. Subagent: `git-expert`

**Role**: Git & Release Engineer  
**Scope**: Version control, staging hygiene, Conventional Commits, branch management, and safe remote pushing.

### Important Operating Policies for `git-expert`:
1. **Explicit Invocation Only**: Do **NOT** invoke `git-expert` automatically after every routine code change. Invoke `git-expert` **ONLY** when the user explicitly requests to commit, push, stage, or release code.
2. **Smart Conventional Commit Types**: Never default to `feat:` blindly. Carefully examine the staged diff and assign the most accurate Conventional Commit type:
   - `fix:` Bug fixes, defect corrections, truncation fixes, crash resolution, regression patches.
   - `style:` Visual polish, CSS/Lip Gloss styles, colors, padding, borders, alignment, whitespace (no functional logic changes).
   - `feat:` New features, new user-facing commands, new components, or major capabilities.
   - `refactor:` Code restructuring that neither fixes a bug nor adds a feature.
   - `perf:` Performance optimizations, query speedups, connection pooling improvements.
   - `test:` Adding or updating unit tests, integration tests, or QA scripts.
   - `docs:` Documentation, README, runbooks, or AGENTS.md updates.
   - `chore:` Build scripts, Makefiles, dependencies, configuration files.

```json
{
  "TypeName": "git-expert",
  "Role": "Git & Release Engineer",
  "Prompt": "Review the staged styling changes, craft a Conventional Commit using style: or fix: as appropriate, and push to origin main."
}
```



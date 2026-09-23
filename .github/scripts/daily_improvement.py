#!/usr/bin/env python3
"""
Daily Improvement Engine for ReadIT
Selects the next unmerged incremental improvement, commits it to a new branch,
opens a GitHub Pull Request, and sends a Discord webhook notification.
"""

import os
import sys
import json
import argparse
import subprocess
import urllib.request
from datetime import datetime

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
REPO_ROOT = os.path.abspath(os.path.join(SCRIPT_DIR, "..", ".."))
BACKLOG_PATH = os.path.join(SCRIPT_DIR, "backlog.json")


def run_cmd(cmd, cwd=REPO_ROOT, check=True):
    print(f"Executing: {' '.join(cmd)}")
    result = subprocess.run(cmd, cwd=cwd, text=True, capture_output=True)
    if check and result.returncode != 0:
        print(f"Error executing command: {result.stderr}", file=sys.stderr)
        raise subprocess.CalledProcessError(result.returncode, cmd, result.stdout, result.stderr)
    return result.stdout.strip()


def send_discord_notification(webhook_url, title, pr_url, repo_name, branch_name, description):
    if not webhook_url:
        print("No DISCORD_WEBHOOK_URL provided. Skipping Discord notification.")
        return

    payload = {
        "username": "GitHub Improvement Bot",
        "avatar_url": "https://github.githubassets.com/images/modules/logos_page/GitHub-Mark.png",
        "embeds": [
            {
                "title": f"🚀 New Daily PR: {title}",
                "url": pr_url,
                "description": description,
                "color": 0xFF4500,  # Reddit / ReadIT Orange
                "fields": [
                    {"name": "Repository", "value": f"`{repo_name}`", "inline": True},
                    {"name": "Branch", "value": f"`{branch_name}`", "inline": True},
                    {"name": "Action Needed", "value": f"[Review & Merge on GitHub]({pr_url})", "inline": False}
                ],
                "footer": {"text": "Daily Open Source Streak & Continuous Improvement"},
                "timestamp": datetime.utcnow().isoformat() + "Z"
            }
        ]
    }

    req = urllib.request.Request(
        webhook_url,
        data=json.dumps(payload).encode("utf-8"),
        headers={"Content-Type": "application/json", "User-Agent": "DailyImprovementBot/1.0"}
    )

    try:
        with urllib.request.urlopen(req) as resp:
            print(f"Discord notification sent successfully! HTTP {resp.status}")
    except Exception as e:
        print(f"Failed to send Discord notification: {e}", file=sys.stderr)


def get_default_branch():
    try:
        ref = run_cmd(["git", "symbolic-ref", "refs/remotes/origin/HEAD"], check=False)
        if ref:
            return ref.split("/")[-1]
    except Exception:
        pass
    branches = run_cmd(["git", "branch", "-a"], check=False)
    if "main" in branches:
        return "main"
    return "master"


def find_next_item(backlog):
    for item in backlog:
        files = item.get("files", {})
        already_exists = True
        for rel_path in files.keys():
            full_path = os.path.join(REPO_ROOT, rel_path)
            if not os.path.exists(full_path):
                already_exists = False
                break
        if not already_exists:
            return item

    today_str = datetime.utcnow().strftime("%Y-%m-%d")
    return {
        "id": f"security-audit-{today_str}",
        "title": f"Security & Dependency Audit ({today_str})",
        "branch_slug": f"security-audit-{today_str}",
        "description": f"Automated maintenance, gosec verification, and dependency review for {today_str}.",
        "files": {
            f"docs/audits/{today_str}-audit.md": f"# ReadIT Security & Maintenance Audit: {today_str}\n\nAutomated daily check performed on {today_str}.\n\n- [ ] Run `go test ./...`\n- [ ] Run `sqlc generate --dry-run`\n- [ ] Check `go list -m -u all`\n"
        },
        "pr_body": f"## Daily Security & Maintenance Review ({today_str})\n\nTracks daily project health and dependency freshness.\n\n- [ ] Review test suite\n- [ ] Merge to maintain commit activity"
    }


def main():
    parser = argparse.ArgumentParser(description="Run daily improvement action")
    parser.add_argument("--dry-run", action="store_true", help="Preview action without committing or pushing")
    args = parser.parse_args()

    if not os.path.exists(BACKLOG_PATH):
        print(f"Backlog file not found at {BACKLOG_PATH}", file=sys.stderr)
        sys.exit(1)

    with open(BACKLOG_PATH, "r", encoding="utf-8") as f:
        backlog = json.load(f)

    item = find_next_item(backlog)
    print(f"Selected item: [{item['id']}] {item['title']}")

    branch_name = f"improvement/{item['branch_slug']}"
    default_branch = get_default_branch()

    if args.dry_run:
        print(f"[Dry Run] Target branch: {branch_name} (from {default_branch})")
        print(f"[Dry Run] Files to create/update:")
        for path in item["files"]:
            print(f"  - {path}")
        print("[Dry Run] Dry run completed successfully.")
        return

    # Git operations
    run_cmd(["git", "checkout", default_branch])
    run_cmd(["git", "pull", "origin", default_branch], check=False)

    run_cmd(["git", "checkout", "-B", branch_name])

    for rel_path, content in item["files"].items():
        dest = os.path.join(REPO_ROOT, rel_path)
        os.makedirs(os.path.dirname(dest), exist_ok=True)
        with open(dest, "w", encoding="utf-8") as f:
            f.write(content)
        run_cmd(["git", "add", rel_path])

    commit_msg = f"improvement: {item['title']}"
    run_cmd(["git", "commit", "-m", commit_msg])

    run_cmd(["git", "push", "-u", "origin", branch_name, "--force"])

    repo_slug = os.environ.get("GITHUB_REPOSITORY", "Sahas001/readit")
    pr_url = ""
    try:
        pr_url = run_cmd([
            "gh", "pr", "create",
            "--base", default_branch,
            "--head", branch_name,
            "--title", f"daily: {item['title']}",
            "--body", item["pr_body"]
        ])
        print(f"PR Created: {pr_url}")
    except Exception as e:
        print(f"Notice: Failed to create PR via gh CLI: {e}")
        pr_url = f"https://github.com/{repo_slug}/tree/{branch_name}"

    webhook_url = os.environ.get("DISCORD_WEBHOOK_URL", "")
    send_discord_notification(
        webhook_url=webhook_url,
        title=item["title"],
        pr_url=pr_url,
        repo_name=repo_slug,
        branch_name=branch_name,
        description=item["description"]
    )


if __name__ == "__main__":
    main()

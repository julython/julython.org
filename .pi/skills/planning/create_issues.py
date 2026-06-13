#!/usr/bin/env python3
"""Helper for creating planning issues with special characters in bodies.

Usage: python3 create_issues.py --repo org/repo --milestone N issues.json

Where issues.json is an array of {"title": "...", "body": "..."} objects.
"""

import json, subprocess, sys, os, argparse, tempfile


def create_issue(repo, title, body, milestone):
    issue = {
        "title": title,
        "body": body,
        "milestone": milestone,
    }
    body_str = json.dumps(issue, ensure_ascii=False)
    with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as tf:
        tf.write(body_str)
        body_path = tf.name
    
    cmd = [
        "gh", "api", f"repos/{repo}/issues", "-X", "POST",
        "--input", body_path,
    ]
    result = subprocess.run(cmd, capture_output=True, text=True)
    os.unlink(body_path)
    if result.returncode != 0:
        print(f"ERROR: {result.stderr}", file=sys.stderr)
        return None, None
    data = json.loads(result.stdout)
    return data.get("number"), data.get("html_url")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Create planning issues via GitHub API")
    parser.add_argument("--repo", default="julython/julython.org", help="GitHub repo (org/name)")
    parser.add_argument("--milestone", type=int, required=True, help="Milestone number")
    parser.add_argument("issues.json", help="JSON file with issues array")
    args = parser.parse_args()
    
    with open(args.issues.json) as f:
        issues = json.load(f)
    
    for issue in issues:
        num, url = create_issue(args.repo, issue["title"], issue.get("body", ""), args.milestone)
        if num:
            print(f"Created #{num}: {url}")
        else:
            print(f"Failed: {issue['title']}")

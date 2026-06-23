---
title: "Scoring Updates"
date: "2026-04-06"
slug: "scoring-2026"
blurb: "How do I win this thing?"
---

## Scoring History

Julython was started in 2012 and was loosely based on the now defunct National Novel Writing Month, which encourages participants to write 50,000 words to "win". We chose to track scoring by tracking commits via a webhook, giving participants 1 point for each commit and 10 points for a new repo added to the game. This was acceptable as a means to track users, so if you were playing "fairly" it was just for your own personal tracking - a gentle nudge to keep committing each day. While this is very easy to track, it doesn't really fit with the goal of helping people learn or actually ship meaningful code.

## What About AI?

As you know, a lot has changed since 2012 and many people are predicting doom and gloom for the software development industry. While we can't say for certain what day-to-day work will look like, we can say that software development best practices and good software design are now more important than ever - regardless of whether you are physically writing the code yourself.

What does that mean for Julython?

We actually need to get better at design and documentation in order to develop software in this new landscape. In effect, we are getting closer to the problem that NaNoWriMo originally attempted to tackle: you have an idea for a tool or want to get involved in software development, but the path forward is still daunting. Sure, anyone can prompt Claude to "write a first person shooter game" and it may even produce something playable. But it won't be great or easy to maintain. You'll constantly be prompting the LLM to fix things or add features that introduce bugs or break the app entirely.

With our new scoring we attempt to address these issues by tracking the health of your project across a few key metrics. Completing all the metrics won't magically make your software better, but it will help you as you refactor and grow. The only constant in programming is that things always change - so be prepared.

The best analogy for the period we're in: "Civil engineers and architects draw the plans for a new building, then hand it off to a team of workers who actually do the work." The key difference with AI is that labor is incredibly fast and cheap. The cost of writing greenfield software is racing toward zero, but the costs associated with maintaining it have not changed - and left unchecked, will only get worse.

## How scoring works

We care a lot about [your privacy](/privacy) and don't want to hand code and IP to "Big Tech" for optional features. Julython has always been about open source, and that has not changed.

When a **push event fires on your repository's main (or default) branch**, our server scans the code from GitHub and evaluates eight metrics using simple heuristics - no AI, no LLM, no guesses. Each metric is scored 0–10 based on what it finds: a README that's substantial enough, test files in standard folders, CI configuration, dependency management, and so on.

The scan runs automatically. For **public** repositories, the server fetches the repo tree and file contents from GitHub to score against. For **private** repositories, no server-side analysis runs - you get zero points from those. So yes - for public repos, analysis data is pulled server-side from GitHub.

### Levels (0–3): how deep goes your project

The L1 scan assigns each metric a **score** (0–10) based on heuristic signals. But a metric with a partial score shouldn't count the same as a fully complete one. That's where **levels** come in.

**Level 1** means the metric has at least one positive signal from the L1 scan (score > 0). Once you have any L1 evidence for a metric, you can **claim a higher level** - L2 or L3 - by confirming additional quality markers in your code. Higher levels multiply your L1 score:

| Level | Multiplier | Meaning |
|---|---|---|
| 0 | — | No analysis row (never scanned) |
| 1 | × 2 | Basic signals present (L1 scan found something) |
| 2 | × 4 | Quality confirmed - you've verified deeper markers |
| 3 | × 6 | High quality - all quality markers met |

**Board points** combine **score** (0–10) and **level** (0–3) per metric: **points = score × level × 2** per metric, capped at **60** per metric when score is 10 and level is 3. Across **8** metrics, a repo can earn up to **480** points; up to **3** repos per game → **1,440** points max per player.

## Browser LLM (experimental)

We really are into 'free' LLM's over here and have experimented by putting a chat box on the project page. This runs entirely in your browser so we don't have to pay for it (we are cheap). But you can ask targeted questions and we'll send a bit of metadata about your project to your browser. This will allow the LLM that has a very limited context window to answer some basic questions about what you need to do to increase a certain score.

## The Metrics

These are the Level 1 checks we have defined so far. The list will evolve over time - we'll adjust thresholds as we learn what signals actually matter.

| Metric | Key Signals |
|---|---|
| **README** | Substantial README with install/getting started and usage instructions; badges like build status |
| **Tests** | Tests in standard folders (e.g. `tests/`); multiple test files; standard framework; test coverage reported |
| **CI** | CI system in use; lint / test / build steps defined |
| **Structure** | Code organized properly or documented; `.gitignore` present; LICENSE present |
| **Linting** | Linting configured; pre-commit or similar tool set up |
| **Dependencies** | Dependencies declared in a standard file; Dependabot or Renovate configured; lock file present |
| **Documentation** | `docs/` folder; CHANGELOG; CONTRIBUTING guide; ARCHITECTURE document; Architecture Decision Records (ADRs) |
| **AI Readiness** | Agent rules or `AGENTS.md`; succinct rules with longer detail in separate docs; clear module boundaries and inline comments for LLM navigation |

Each time you run an analysis we'll update the scores, reflecting them on your game board. Level 1 checks can change over time, so scores can go down as well as up. For example, if we raise the required coverage threshold to 70% and your ratio drops after a big refactor, you'll lose those points until you bring it back up.

## Scoring Limits

Going forward we are limiting players to **3 repos per game**. Only the owner (or an admin) can trigger a rescan of the L1 analysis for that project.

With any competition there will be attempts to game the system. We'll be looking at ways to verify scores are legitimate - most likely through some form of community review for players who are making things less fun for everyone else.

## But How Do I Win?

That's really up to you. If you learn something or make your projects better, that's winning in our minds. Competition is one major driver, so we plan to add other ways to score points - some ideas include bonus points for joining teams or achieving high L2/L3 levels on metrics. Nothing concrete yet, but stay tuned.

We're also thinking about **low-cost ways** to use open-source LLMs to enhance scoring - possibly leveraging edge LLMs running in the browser, similar to the existing chat feature. If we ever do bring AI into scoring it'll be browser-side only, respecting the same privacy-first philosophy we've built Julython on.

Stay tuned for more details!

-- The Julython Team

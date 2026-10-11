# PRD: Grumble (working name)

A public, crowdsourced board of real frustrations, ranked by urgency and willingness to pay, with a paid layer for founders.

| | |
| --- | --- |
| Status | Draft v0.1 |
| Owner (product) | TBD |
| Owner (engineering) | TBD |
| Last updated | 2026-10-10 |
| Reviewers | TBD |
| Companion docs | `ARCHITECTURE.md` (how it is built), `CLAUDE.md` (agent instructions) |

"Grumble" is a placeholder. Rename before launch, and do not reuse the name, copy or branding of the reference product (see section 2).

---

## 1. Summary

### 1.1 Problem

Founders pick problems to solve from guesses, Reddit threads and a few interviews, and have no cheap way to see how often a problem hurts, how badly, and whether people would pay to fix it. People who are frustrated have no low-friction place to say so and see that others feel the same.

### 1.2 Proposed solution

A web product with two sides built on one dataset:

- **Venters** post a frustration in one sentence with no signup, vote on others, and see where in the world and in which categories pain clusters.
- **Builders** read a ranked, filterable, exportable view of the same data, with an urgency score that combines what the submitter says (frequency, disruption, willingness to pay) with how the crowd votes. A paid tier adds a high-signal feed, AI-written briefs, exports and alerts.

### 1.3 Why now

AI-assisted building has lowered the cost of shipping software, so the scarce input is validated demand, not code. Search-indexed, human-written complaint data is a durable acquisition channel for that audience.

---

## 2. Reference product and what we change

This product is modelled on a public site, PainMap (worldpainmap.com), reviewed on 2026-10-09 from its public pages only. We reproduce the concept, not the content, name, copy or visual identity. Observed behaviours we keep, and defects we fix:

| Observed in the reference | Decision here |
| --- | --- |
| Anonymous posting: one text box (max 2000 chars), category, country | Keep (FR-SUB-1 to 4) |
| Optional "more details" form for frequency, disruption, willingness to pay | Keep, and make the fields explicit and validated (FR-SUB-5) |
| Urgency score from submitter inputs plus votes; weights unpublished | Keep the idea, publish the formula (section 9) |
| Trending page headed "by urgency" but described as "most upvoted" | One definition per tab (FR-FEED-2) |
| Hub pages show "top 0" titles when empty | Never render counts of zero in titles; noindex until 5 pains (FR-SEO-3) |
| Digest promised on Sunday, with 3 picks in one place, 5 in another, Monday in a third | One schedule (FR-EMAIL-2) |
| Pro price differs between pages | One price source of truth in config (FR-PRO-1) |
| Copy says no AI, product sells AI briefs | Say exactly: pains are human-written, briefs are AI-generated and labelled (FR-PRO-3) |
| No warning about personal data in free text | PII detection and warning (FR-MOD-3) |
| Several pages rendered empty without scripts | Server-render every list (NFR-SEO-1) |

---

## 3. Goals and success metrics

Targets are proposals to validate with the owner, measured from public launch (L).

| Goal | Metric | Baseline | Target | Source | Review |
| --- | --- | --- | --- | --- | --- |
| G1 Supply of pains | Published pains | 0 | 1,000 by L+90d | DB count | Weekly |
| G2 Contribution is easy | Submit conversion (visit /submit to published pain) | n/a | 35% | Analytics funnel | Weekly |
| G3 Trust in content | Share of pains rejected or removed after publish | n/a | under 3% | Moderation log | Weekly |
| G4 Organic growth | Monthly sessions from search | 0 | 20,000 by L+120d | Search Console | Monthly |
| G5 Engagement | Median votes per published pain at 7 days | n/a | 3 | DB | Weekly |
| G6 Revenue | Paying Pro subscribers | 0 | 100 by L+180d | Billing provider | Monthly |
| G7 Quality of signal | Pains with complete detail fields (frequency, disruption, pay) | n/a | 40% | DB | Monthly |
| G8 Speed | Time from submit to visible | n/a | p95 under 15 s | Traces | Weekly |

Guardrails: p75 Largest Contentful Paint under 2.5 s on mobile; moderation backlog under 24 h; zero exposure of personal identifiers.

---

## 4. Non-goals (v1)

- Native mobile apps.
- Threaded discussion, direct messages, follows, or profiles with public handles.
- Generating pains with AI. Every pain is human-written.
- Scraping Reddit or other sites to seed content as if users wrote it. Seed content, if any, is labelled.
- User-to-user marketplace or paid bounties for solving a pain.
- Languages other than English at launch (data model must not block it, NFR-I18N-1).
- Selling or sharing raw submissions with third parties.

---

## 5. Personas

| Persona | Needs | Success looks like |
| --- | --- | --- |
| Vera, the venter | Say it in under a minute, no account, feel less alone | Posts, sees votes appear, shares the link |
| Farid, the indie founder | A shortlist of software-fixable pains with willingness to pay | Exports 10 pains, picks one, returns weekly |
| Rin, the researcher or journalist | Slice by country and topic, cite stable URLs | Finds a country trend, links to a hub page |
| Moe, the moderator (admin) | Fast queue, clear reasons, undo | Clears the queue daily, reverses mistakes |

---

## 6. Release plan

| Milestone | Scope | Exit criteria |
| --- | --- | --- |
| M0 Foundation (week 1) | Repo scaffold, CI, database schema, taxonomies, design tokens | Empty app deploys; migrations run; CI green |
| M1 Core loop (weeks 2-4) | Submit, moderation pipeline, feed, vote, pain detail, hubs, SEO basics | A stranger can post, see it live, and vote |
| M2 Discovery (weeks 5-6) | Trending, map, search, comments, flags, bookmarks, email digest | Map and trending driven by live data |
| M3 Monetization (weeks 7-9) | Accounts, Pro, Money feed, AI briefs, exports, alerts, Paddle billing | A test purchase unlocks Pro end to end |
| M4 Launch (weeks 10-11) | Seed content, legal pages, performance and a11y pass, load test, launch checklist | Section 17 checklist complete |

---

## 7. User stories and acceptance criteria

### Contribute

- **US-01** As Vera, I post a frustration without an account.
  - Given I am on /submit, when I enter at least 20 characters, choose a category and country, and submit, then my pain appears in the feed within 15 s (or I am told it is under review).
  - The form never asks for email, name or password.
  - Text over 2000 characters is blocked with a visible counter.
- **US-02** As Vera, I add detail so founders can judge urgency.
  - I can set frequency (daily, weekly, sometimes), disruption (mild, annoying, ruins my week), whether I would pay, an optional monthly amount in USD, and what I already tried (max 500 characters).
  - These are optional; skipping them still publishes the pain.
- **US-03** As Vera, I am warned before I publish personal data.
  - If the text contains an email, phone number or a government ID pattern, I see a warning and must edit or confirm redaction before submitting.

### Discover and react

- **US-04** As any visitor, I browse the feed and switch between Trending, Latest and Most voted, and filter by category, with server-rendered results.
- **US-05** As any visitor, I vote on a pain once, and see the count change instantly; repeating the vote removes it.
- **US-06** As any visitor, I open the map, pick a category or region, click a country, and see its top pains.
- **US-07** As any visitor, I open a pain's own page with its text, votes, category, country, comments and related pains.
- **US-08** As any visitor, I flag a pain as spam, abuse or personal data.
- **US-09** As any visitor, I bookmark pains (5 without a plan, unlimited on Pro).

### Build

- **US-10** As Farid, I sign in, subscribe to Pro, and open the Money feed of pains with urgency 80 or more where submitters said they would pay.
- **US-11** As Farid, I open an AI brief for any pain (who it is for, why it hurts, what was tried, what to build, or a skip verdict), clearly labelled as AI-generated.
- **US-12** As Farid, I export the current filtered view as CSV or PDF (up to 500 rows).
- **US-13** As Farid, I create an alert for a keyword or category and receive an email and in-app notice when a matching pain crosses my threshold.

### Operate

- **US-14** As Moe, I review held and flagged pains in one queue, approve, reject with a reason, or restore, and every action is logged.
- **US-15** As an owner, I see submission volume, moderation backlog, conversion and revenue on one admin page.

---

## 8. Functional requirements

Priority uses MoSCoW. Milestone is the first release that ships it.

### Submission

| ID | Requirement | Pri | M |
| --- | --- | --- | --- |
| FR-SUB-1 | Free-text pain, 20 to 2000 characters, plain text only (no HTML or markdown rendering) | Must | M1 |
| FR-SUB-2 | Category required, one of the 22 in the appendix | Must | M1 |
| FR-SUB-3 | Country required, ISO 3166-1 list plus "Other" | Must | M1 |
| FR-SUB-4 | No account, email or name required to submit | Must | M1 |
| FR-SUB-5 | Optional detail: frequency, disruption, would-pay flag, monthly amount, what was tried | Must | M1 |
| FR-SUB-6 | Category suggestion from text when none chosen (user can override) | Could | M2 |
| FR-SUB-7 | Duplicate detection: near-duplicate of an existing pain prompts "vote on this instead" | Should | M2 |
| FR-SUB-8 | Unique, stable, human-readable slug per pain | Must | M1 |

### Moderation and trust

| ID | Requirement | Pri | M |
| --- | --- | --- | --- |
| FR-MOD-1 | New pains start as `pending`; automated checks decide `published`, `held` or `rejected` | Must | M1 |
| FR-MOD-2 | Rule checks: length, link count over 2, blocked terms, repeated text, per-source rate limits | Must | M1 |
| FR-MOD-3 | Personal data detection (email, phone, ID patterns) with user warning and automatic redaction on confirm | Must | M1 |
| FR-MOD-4 | Model-based classification for spam, abuse, self-harm, personal data; low confidence goes to `held` | Should | M2 |
| FR-MOD-5 | Three flags from distinct sources auto-hide a pain pending review | Must | M2 |
| FR-MOD-6 | Admin queue with approve, reject (reason required), restore, and audit log | Must | M2 |
| FR-MOD-7 | Cloudflare Turnstile (or equivalent) on submit and flag | Must | M1 |
| FR-MOD-8 | Self-harm content routes to a safe-handling path: held, never shown publicly, with a support-resources notice to the author | Must | M2 |

### Voting, comments, bookmarks

| ID | Requirement | Pri | M |
| --- | --- | --- | --- |
| FR-VOTE-1 | One vote per pain per identity (signed-in user, else anonymous id); toggle to remove | Must | M1 |
| FR-VOTE-2 | Optimistic UI, authoritative count from the database | Must | M1 |
| FR-VOTE-3 | Per-source vote rate limits and velocity anomaly detection | Must | M2 |
| FR-COM-1 | Comments, 1 to 1000 characters, same moderation pipeline, no threading | Should | M2 |
| FR-BKM-1 | Bookmarks: 5 without a plan, unlimited on Pro; anonymous bookmarks merge on sign-in | Must | M2 |

### Discovery

| ID | Requirement | Pri | M |
| --- | --- | --- | --- |
| FR-FEED-1 | Explore feed with search, category chips, tabs Trending, Latest, Most voted | Must | M1 |
| FR-FEED-2 | Tab definitions are fixed: Latest = newest first; Most voted = vote count; Trending = hot rank (section 9.3) | Must | M1 |
| FR-FEED-3 | Trending page: top 50 by urgency over the last 7 days, plus top countries and categories | Must | M2 |
| FR-FEED-4 | Live counters (pains, countries, votes) updating without reload | Should | M2 |
| FR-MAP-1 | Country choropleth by published-pain count with category and region filters | Must | M2 |
| FR-MAP-2 | Click a country to list its top pains; keyboard accessible with a text list fallback | Must | M2 |
| FR-HUB-1 | Hub pages by category, country and keyword, each with live count, ranked list, original explanatory text and FAQ | Must | M1 |
| FR-HUB-2 | Hub index at /pain/hubs linking every hub | Should | M2 |
| FR-DET-1 | Pain detail page with related pains and share controls | Must | M1 |

### Accounts, Pro and email

| ID | Requirement | Pri | M |
| --- | --- | --- | --- |
| FR-ACC-1 | Sign in by email magic link and Google; anonymous use remains fully supported | Must | M3 |
| FR-ACC-2 | Account export and deletion on request, fulfilled within 30 days | Must | M3 |
| FR-PRO-1 | Plans and prices defined once in `internal/domain/billing/plans.go`; all pages read from it | Must | M3 |
| FR-PRO-2 | Money feed: urgency 80 or more, would-pay true, frequency daily or weekly | Must | M3 |
| FR-PRO-3 | AI Founder Brief generated on demand, cached per pain, labelled "AI-generated", grounded only in stored signals | Must | M3 |
| FR-PRO-4 | CSV (up to 500 rows) and PDF export of the current filtered view | Should | M3 |
| FR-PRO-5 | Alerts on keyword, category or pain thresholds, by email and in-app | Should | M3 |
| FR-PRO-6 | Billing through a Merchant of Record; entitlements updated only from verified webhooks | Must | M3 |
| FR-EMAIL-1 | Newsletter sign-up with double opt-in and one-click unsubscribe | Must | M2 |
| FR-EMAIL-2 | Free digest: Sunday 08:00 UTC, top 3 pains of the week. Pro digest: Monday 08:00 UTC, 5 picks with briefs | Should | M3 |

### SEO and sharing

| ID | Requirement | Pri | M |
| --- | --- | --- | --- |
| FR-SEO-1 | Server-rendered pages with canonical URL, unique title and description | Must | M1 |
| FR-SEO-2 | Generated Open Graph image per page type | Should | M2 |
| FR-SEO-3 | Hubs and pain pages indexable only with at least 5 published pains (hubs) or published status (pains); titles never show zero counts | Must | M1 |
| FR-SEO-4 | XML sitemap of indexable URLs; JSON-LD on pain pages | Should | M2 |

### Admin

| ID | Requirement | Pri | M |
| --- | --- | --- | --- |
| FR-ADM-1 | Role-based admin area (owner, moderator) | Must | M2 |
| FR-ADM-2 | Metrics page: volume, backlog, conversion, revenue | Should | M3 |

---

## 9. Scoring specification

These are design defaults. Weights are tunable in `internal/scoring/weights.go` and every change is a versioned, logged decision (see ADR process in `ARCHITECTURE.md`).

### 9.1 Inputs

| Input | Source | Normalised value |
| --- | --- | --- |
| Frequency | Submitter | daily 1.00, weekly 0.60, sometimes 0.30, unset 0.30 |
| Disruption | Submitter | ruins my week 1.00, annoying 0.55, mild 0.20, unset 0.35 |
| Willingness to pay | Submitter | no 0, yes without amount 0.60, yes with amount: min(1, 0.6 + 0.4 x amount/20), amount in USD per month |
| Crowd | Votes | min(1, ln(1 + votes) / ln(1 + 50)) |

### 9.2 Urgency score (0 to 100)

```
urgency = round(100 * (0.30*frequency + 0.25*disruption + 0.25*pay + 0.20*crowd))
```

Unset submitter fields use the neutral values above, so a one-sentence pain still scores. Example: daily, annoying, would pay $10, 10 votes gives 76; daily, ruins my week, would pay $20 or more, 50 votes gives 100. The Money feed threshold (80) therefore needs strong submitter signal and crowd support.

### 9.3 Trending (hot rank)

```
hot = (votes_7d + 0.5*comments_7d + urgency/20) / (age_hours + 2)^1.5
```

### 9.4 Anti-gaming

- Crowd contributes at most 20 points and is log-scaled, so vote floods move the score little.
- Submitter fields are self-reported; the score is displayed with its inputs on hover so readers can discount it.
- Votes from the same source cluster (hashed network, anonymous id) count once after the first five.

---

## 10. Non-functional requirements

| ID | Requirement |
| --- | --- |
| NFR-PERF-1 | Mobile p75 LCP under 2.5 s, INP under 200 ms, CLS under 0.1 on feed, hub and pain pages |
| NFR-PERF-2 | Vote and submit API p95 under 400 ms excluding model calls |
| NFR-SEO-1 | Every list and hub page returns its content in the initial HTML |
| NFR-A11Y-1 | WCAG 2.2 AA; map has a keyboard path and a text equivalent; colour is never the only signal |
| NFR-SEC-1 | Clients never write to the database directly; all writes pass server handlers with input validation |
| NFR-SEC-2 | No RLS; the database is private to the app. Public reads limited to `status = 'published'` by dedicated repository methods and the `published_pains` view, proven by tests |
| NFR-SEC-3 | Secrets only in environment variables; webhooks verify signatures; admin routes require role and re-auth |
| NFR-SEC-4 | Output encoding everywhere user text is rendered; no raw HTML from users, ever |
| NFR-PRIV-1 | Store no raw IP addresses beyond a 24 h rate-limit window; keep only a salted hash for abuse analysis, rotated monthly |
| NFR-PRIV-2 | Cookie banner and consent for ads and non-essential analytics in the EEA and UK; essential cookies documented |
| NFR-PRIV-3 | Deletion removes account data; authored pains are anonymised, not left attributable |
| NFR-REL-1 | 99.5% monthly availability target; database backups daily with tested restore |
| NFR-SCALE-1 | Design for 100,000 pains and 1,000,000 votes without architecture change |
| NFR-I18N-1 | All user-facing strings externalised; dates and numbers locale-aware; slugs unicode-safe |
| NFR-OBS-1 | Structured logs, error tracking, uptime check, and a dashboard for queue depth and p95 latency |

---

## 11. Analytics events

Names are `snake_case`; no personal data in properties.

| Event | Properties |
| --- | --- |
| `submit_started`, `submit_completed`, `submit_blocked` | mode (quick, detailed), category, country, reason |
| `vote_cast`, `vote_removed` | pain_id, source (feed, hub, detail) |
| `feed_filtered` | tab, category, search_used |
| `map_country_opened` | country, category |
| `flag_submitted` | reason |
| `pro_viewed`, `checkout_started`, `subscription_activated` | plan |
| `brief_opened`, `export_downloaded`, `alert_created` | format or type |
| `digest_opened`, `digest_clicked` | edition |

---

## 12. Monetization (proposed, validate)

| Item | Proposal |
| --- | --- |
| Free | Browse, submit, vote, comment, 5 bookmarks, ads |
| Pro | Money feed, full AI briefs, CSV and PDF export, alerts, Founder Mode filter, unlimited bookmarks, no ads. Proposed 12 USD per month or 99 USD per year, one-month trial |
| Founder Pack | One-time purchase: CSV of top 500 pains, 3 briefs, read-only feed. Proposed 49 USD |
| Refunds | 14 days, no questions |
| Provider | A Merchant of Record handles tax and invoices |
| Ads | Optional, off at launch until traffic justifies consent-management cost |

---

## 13. Trust and safety policy

- Allowed: first-person frustrations about products, services, institutions and situations.
- Not allowed: naming private individuals, personal data, harassment, hate, sexual content, illegal content, advertising, and instructions for harm.
- Self-harm or crisis content is never published; the author sees a supportive notice with regional resources.
- Defamation risk: pains naming a company are allowed when they describe the author's own experience; takedown requests are handled via the contact page within 7 days.
- Every moderator action is logged with actor, reason and timestamp, and reversible for 30 days.

---

## 14. Dependencies, assumptions and risks

| # | Item | Type | Mitigation or owner |
| --- | --- | --- | --- |
| 1 | Empty board at launch looks dead | Risk | Seed 150 to 300 pains written by the team or recruited contributors, labelled "seeded" in data, never presented as organic |
| 2 | Spam and abuse on anonymous posting | Risk | Turnstile, rate limits, moderation pipeline, flag auto-hide |
| 3 | Search traffic is slow to arrive | Risk | Hub pages with original text; sitemap; internal linking; accept 3 to 6 months |
| 4 | Self-reported scores are gameable | Risk | Section 9.4; show inputs; consider verified-purchase signals later |
| 5 | Defamation or personal data complaints | Risk | Policy in section 13; PII detection; fast takedown |
| 6 | AI briefs invent facts | Risk | Ground only in stored fields; refuse if fewer than 3 signals; label as AI |
| 7 | Reference product copied too closely | Risk | Original name, copy, visual design; review before launch |
| 8 | Dependency on third-party services (database, email, payments, model API) | Dependency | Abstraction per provider in `internal/platform`, documented fallbacks |
| 9 | Data volume assumptions are unverified | Assumption | Revisit after M1 with real traffic |
| 10 | Merchant of Record availability for the owner's country | Assumption | Confirm before M3 |

---

## 15. Open questions

| # | Question | Owner | Needed by |
| --- | --- | --- | --- |
| Q1 | Final product name and domain | Owner | M0 |
| Q2 | Are anonymous votes allowed without Turnstile, or only on first vote? | Product | M1 |
| Q3 | Pricing and trial length | Product | M3 |
| Q4 | Ads at launch, or later? | Product | M4 |
| Q5 | Which model tier for moderation versus briefs (cost and quality trade-off)? | Engineering | M2 |
| Q6 | Legal review of terms, privacy, and takedown process | Owner | M4 |
| Q7 | Seed-content approach and labelling | Product | M3 |
| Q8 | Do pain authors get an edit or delete link (via a secret token) after posting? | Product | M2 |

---

## 16. Out-of-scope follow-ups

Public API, webhooks for builders, team seats, verified-buyer signals, translations, native apps, "claim and respond" pages for companies.

---

## 17. Launch checklist

- [ ] All Must requirements for M1 to M3 pass acceptance criteria
- [ ] Lighthouse mobile: performance 90+, accessibility 95+, SEO 100 on feed, hub, pain pages
- [ ] Load test: 200 requests per second on feed and vote for 10 minutes within targets
- [ ] Restore from backup rehearsed
- [ ] Published-only read tests pass; unpublished rows never appear in any public read or export
- [ ] Terms, privacy, refunds, contact and takedown pages live
- [ ] Consent banner tested in EEA and UK settings
- [ ] Sitemap submitted; robots and noindex rules verified
- [ ] Moderation queue staffed; on-call contact for incidents
- [ ] Seed content labelled and reviewed

---

## 18. Change history

| Date | Author | Change |
| --- | --- | --- |
| 2026-10-10 | product | First draft from review of the reference product and best-practice PRD structure |
| 2026-10-10 | engineering | Align to the Go build: plans path to `internal/domain/billing/plans.go`, weights path to `internal/scoring/weights.go`, drop RLS in favour of published-only repository reads, provider abstraction under `internal/platform` |

---

## Appendix A. Categories (22)

Business, Health, Education, Housing, Finance, Travel, Tech, Relationships, Food, Government, Sports, Entertainment, Gaming, Parenting, Pets, Fitness, Fashion, Transportation, Environment, Career, Shopping, Other.

## Appendix B. Launch keyword hubs (19)

Parking, Rent and landlords, Insurance, Taxes, Dating apps, Commute and traffic, Wait times, Scams and fraud, Billing and invoices, Customer support, Passwords and logins, Shipping and delivery, Doctors and clinics, Schools, Remote work, Mental health, Startup ideas, Customer pain points, Small business problems.

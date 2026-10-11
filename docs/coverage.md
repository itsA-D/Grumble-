# Requirement coverage

Tracks which `PRD.md` requirement IDs are implemented, and the test or file that
proves each one. Status values: **Done**, **Partial** (some of the requirement is
missing, noted), **Not implemented**. IDs without a test cite the implementing
file; IDs marked Partial that are intentionally deferred also name the M2 gap.
This file grows as milestones land; the M1 gate report is its first snapshot.

## M1 — core loop

| ID | Requirement | Status | Proven by (test or file) |
|---|---|---|---|
| FR-SUB-1 | Free text 20–2000 chars, plain text only | Done | `internal/domain/pain/pain_test.go` (`TestValidateOK`, `TestValidateFailures`); `internal/transport/web/handlers_test.go` (`TestFeedRendersAndEscapesUserText`); `e2e/security.spec.ts` |
| FR-SUB-2 | Category required, one of 22 | Done | `internal/domain/taxonomy/taxonomy_test.go` (`TestCategoryCount`, `TestCategoriesValidAndUnique`); `handlers_test.go` (`TestSubmitFormHasLabelledInputs`) |
| FR-SUB-3 | Country required, ISO 3166-1 + Other | Done | `taxonomy_test.go` (`TestCountries`); `handlers_test.go` (`TestSubmitFormHasLabelledInputs`) |
| FR-SUB-4 | No account, email or name required | Done | `internal/transport/web/submit.go`; `handlers_test.go` (`TestSubmitPublishedRedirectsToDetail`) |
| FR-SUB-5 | Optional detail fields | Done | `internal/service/pains/service_test.go` (`TestSubmitComputesUrgency`); `internal/scoring/scoring_test.go` (`TestUrgencyPRDExamples`) |
| FR-SUB-8 | Unique, stable, human-readable slug | Done | `internal/domain/pain/slug_test.go` (`TestSlugify`, `TestSlugifyBounds`); `service_test.go` (`TestUniqueSlugSuffixesOnCollision`) |
| FR-MOD-1 | pending → published / held / rejected | Done | `internal/domain/moderation/rules_test.go`; `service_test.go` (`TestSubmitPublishesCleanPain`, `TestSubmitRejectsBlockedTerm`, `TestSubmitHoldsTooManyLinks`) |
| FR-MOD-2 | length, link count > 2, blocked terms, repeated text, rate limits | Done | `rules_test.go` (`TestEvaluateTooManyLinksHeld`, `TestEvaluateBlockedTermRejected`, `TestEvaluateRepeatedSentenceHeld`); `internal/platform/ratelimit/memory_test.go`; `handlers_test.go` (`TestSubmitRateLimited`) |
| FR-MOD-3 | Personal-data detection with warning and redaction | Done | `internal/domain/pii/pii_test.go`; `handlers_test.go` (`TestSubmitPIIReRendersWithRedactionPreview`); `service_test.go` (`TestSubmitRedactsAfterConfirmation`) |
| FR-MOD-7 | Turnstile on submit and flag | Done | `internal/platform/turnstile/turnstile_test.go`; `handlers_test.go` (`TestSubmitTurnstileForbidden`) |
| FR-VOTE-1 | One vote per pain per identity, toggle | Done | `internal/service/voting/service_test.go`; `internal/store/postgres/repositories_test.go` (`TestVoteUniquenessAndToggle`); `e2e/core-loop.spec.ts` |
| FR-VOTE-2 | Optimistic UI, authoritative count | Done | `service_test.go` (`TestSetIsIdempotent`); `handlers_test.go` (`TestVoteHTMXReturnsPartialAndIsIdempotent`) |
| FR-FEED-1 | Explore feed: search, category chips, tabs | Partial | `internal/transport/web/feed.go`; `internal/service/feed/service_test.go` (search and chips land in item 3) |
| FR-FEED-2 | Fixed tab definitions (latest / most voted / hot) | Done | `internal/scoring/scoring_test.go` (`TestHotDecaysWithAge`); `feed/service_test.go` (`TestTrendingRanksVotesThenAge`) |
| FR-HUB-1 | Hubs by category, country and keyword | Partial | `internal/service/hubs/service_test.go`; `internal/transport/web/hubs.go` (keyword hubs deferred) |
| FR-DET-1 | Detail page with related pains and share controls | Partial | `internal/transport/web/views/detail.templ`; `handlers_test.go` (`TestDetailRendersVoteButton`) (share controls deferred) |
| FR-SEO-1 | Server-rendered canonical URL, unique title and description | Done | `internal/transport/web/seo/seo.go`; `handlers_test.go`; `e2e/core-loop.spec.ts` |
| FR-SEO-3 | Indexable only at 5+ published pains; never counts of zero | Done | `hubs/service_test.go` (`TestCategoryHubIndexableAtFloor`, `TestCategoryHubNoindexBelowFloor`, `TestDescriptionNeverSaysZero`); `handlers_test.go` (`TestHubNoIndexBelowFloor`) |
| NFR-SEO-1 | Every list and hub returns content in the initial HTML | Done | `e2e/core-loop.spec.ts`; `handlers_test.go` (`TestHubNoIndexBelowFloor`) |
| NFR-SEC-1/2/4 | No direct client writes; published-only reads; output encoding | Done | `internal/store/postgres/repositories_test.go` (`TestUnpublishedRowsNeverLeak`); `e2e/security.spec.ts`; `handlers_test.go` (`TestFeedRendersAndEscapesUserText`) |
| NFR-PRIV-1 | No raw IP; salted monthly-rotated source hash | Done | `internal/service/identity/identity_test.go` (`TestSourceHashRotatesMonthly`, `TestSourceHashCanonicalisesIP`); `middleware_test.go` (`TestResolveClientAddrTrustsHeaderOnlyFromTrustedPeer`); `handlers_test.go` (`TestRequestLogsNeverContainRawIP`) |

## M2 IDs referenced by already-landed code

| ID | Requirement | Status | Proven by (test or file) |
|---|---|---|---|
| FR-SUB-6 | Category suggestion from text when none chosen | Not implemented | No code or test yet. Deferred (M2, `Could`). |
| FR-SUB-7 | Near-duplicate prompts "vote on this instead" | Partial | Store query `RecentSubmissionFromSource` in `db/queries/pains.sql` and `internal/store/postgres/pains.go` (`RecentFromSource`), proven by `repositories_test.go` (`TestRecentFromSource`). The submission path does not call it and there is no user-facing "vote on this instead" prompt yet. |
| FR-MOD-8 | Self-harm held, never public, with a support-resources notice | Partial | Held and never published: `internal/domain/moderation/selfharm.go`; `rules_test.go` (`TestSelfHarmIsHeld`, `TestSelfHarmBeatsBlockedAndLinks`, `TestSelfHarmDoesNotMatchInsideWords`); `service_test.go` (`TestSubmitHoldsSelfHarm`); `e2e/security.spec.ts`. The support-resources notice to the author is not implemented (`internal/transport/web/submit.go`, `submittedCopy`) — tracked as M2 (see `ARCHITECTURE.md` self-harm safety-floor note). |

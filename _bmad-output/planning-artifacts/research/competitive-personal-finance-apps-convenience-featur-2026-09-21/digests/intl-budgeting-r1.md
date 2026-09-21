# Digest: intl-budgeting cluster (YNAB, Monarch Money) — round 1

Assistant brief: feature teardown (Q1) + customer voice (Q2) for YNAB and Monarch Money.

## Preliminary fact-check
- {claim: "Intuit shut down Mint on 3/23/2024, directed users to Credit Karma; Monarch Money is widely covered as the de facto Mint successor/beneficiary", source: https://betakit.com/as-intuit-winds-down-mint-financial-planning-app-monarch-money-comes-to-canada/, publisher: BetaKit, accessed: 2026-09-21, confidence: medium-high, class: press}
- {claim: "Monarch's own account of the Mint shutdown and migration guidance", source: https://www.monarch.com/blog/mint-shutting-down, publisher: Monarch Money blog, accessed: 2026-09-21, confidence: medium (self-interested party), class: official-doc/press}
- Caveat: migration-scale numbers (20x, 500K users, $30M ARR) are low-confidence, aggregator-sourced only.

## YNAB — Q1 Feature teardown
- {claim: "Scheduled Transactions auto-drop into the register on a chosen frequency — quick-entry for repeats + reminders for infrequent ones (payee, amount, category, account, frequency)", source: https://support.ynab.com/en_us/scheduled-transactions-a-guide-BygrAIFA9, publisher: YNAB Support, accessed: 2026-09-21, confidence: high, class: official-doc}
- {claim: "Auto-matching: scheduled transaction appears bold in register on due date; bank-imported match gets auto-matched/approved, else needs manual approval", source: same as above, confidence: high, class: official-doc}
- {claim: "Auto-Assign feature auto-assigns budgeted amounts to categories", source: https://support.ynab.com/en_us/auto-assign-a-guide-r1gBNbBJo, confidence: high, class: official-doc}
- {claim: "Android: long-press app icon launches transaction-entry screen directly", source: https://www.ynab.com/blog/pressing-news-ynabs-android-app-has-long-press-functionality, confidence: high, class: official-doc}
- {claim: "Official features page: automatic bank import, Apple Card import, multi-device sync incl. offline, shared budgets up to 6 people, goal/target tracking, spending & net-worth reports/charts, loan calculator, Category Templates, Customizable Views, Mobile Widgets", source: https://www.ynab.com/features, confidence: high, class: official-doc}
- {claim: "No mention of receipt/OCR scanning, rules-based smart categorization, or payee autocomplete on official features page — likely gap or unadvertised", source: https://www.ynab.com/features, confidence: medium (absence-of-evidence), class: official-doc}
- {claim: "'Manage Payees' feature cleans up inconsistently formatted payee names", source: search-summary of YNAB community/support content, confidence: low (not directly fetched), class: aggregator}

## YNAB — Q2 Customer voice
- {claim: "4.8/5 across ~55,000 Apple App Store ratings; 4.7/5 across ~21,500 Google Play reviews", source: apps.apple.com/us/app/ynab/id1010865877 (via aggregator search summary), confidence: medium, class: aggregator}
- {claim: "Top convenience complaint: manual entry is tedious (~1 min/transaction per one user report); falling behind a few days causes habit abandonment", source: search synthesis of community discussion, confidence: low-medium, class: aggregator/user-review}
- {claim: "Manage Payees cleanup described as tedious", confidence: low, class: aggregator}
- {claim: "Auto-categorization doesn't reliably match payees as expected", confidence: low, class: aggregator}
- {claim: "$109/year price + steep learning curve are top Reddit/Trustpilot complaints; many abandon within 2 weeks (unlearning passive-tracking habits)", source: trustpilot.com/review/ynab.com (via synthesis), confidence: low-medium, class: user-review/aggregator}
- {claim: "US bank-sync flakiness via Plaid, esp. smaller credit unions", confidence: low, class: aggregator}
- NOTE: no verbatim review/thread text was directly fetched — all Q2 YNAB claims are search-engine-synthesized themes, not primary quotes.

## Monarch Money — Q1 Feature teardown
- {claim: "Auto-categorizes on import; Transaction Rules keyed on statement text/merchant/amount auto-rename, recategorize, tag, set owner, or hide future matches", source: help.monarch.com/hc/en-us/articles/360048393372-Transaction-Rules (direct fetch 403'd, content via search snippet, consistent across 2 passes), confidence: medium, class: official-doc}
- {claim: "Manual transaction edit can be saved as a rule to auto-apply going forward", source: same article, confidence: medium, class: official-doc}
- {claim: "Recurring calendar auto-detects ~80% of recurring items (subscriptions/bills/transfers/paychecks); may miss variable amount/date ones; manual add supported", source: help.monarch.com/hc/en-us/articles/4890751141908 + monarch.com/blog/track-recurring-bills-and-subscriptions, confidence: medium, class: official-doc}
- {claim: "Auto-categorization across 13,000+ linked institutions; marketed for couples/shared dashboards", source: search synthesis re Reddit/r/personalfinance, confidence: low-medium (institution count unconfirmed on official page), class: aggregator}

## Monarch Money — Q2 Customer voice
- {claim: "Community sentiment described as overwhelmingly positive; praise for couples dashboard + broad auto-categorization coverage; widely recommended top Mint replacement", source: search synthesis of Reddit, confidence: low-medium, class: aggregator}
- {claim: "Cleaner design, stronger investment tracking, better collaboration than Mint; actively maintained vs Mint being neglected pre-shutdown", confidence: low, class: aggregator}
- {claim: "Pricing ($99/yr or $14.99/mo, no free tier after 7-day trial) most-debated; ex-Mint users (used to free) most price-sensitive", source: search synthesis re r/personalfinance 2024 threads, confidence: low-medium, class: aggregator}
- {claim: "Frequent complaint: mislabeled transactions needing manual correction, 'constantly miscategorizing' even with rules set up", source: search synthesis of App Store/Trustpilot/justuseapp, confidence: low-medium, class: aggregator/user-review}
- {claim: "Sync reliability called 'the most common complaint about Monarch — and about most aggregator-based budgeting apps in general', incl. smaller institutions", confidence: low-medium, class: aggregator}
- {claim: "No manual-only entry path — if Plaid connection fails/unsupported or user won't share bank creds, app 'will not work' for them", confidence: low, class: aggregator}
- {claim: "No undo for mistaken category-money reallocation", confidence: low, class: aggregator}
- {claim: "Complaint: no spending-limit setting, no monthly cash-flow picture", source: balancepro.app/blog/why-i-stopped-using-monarch-money (title only, not fetched), confidence: low, class: aggregator

## Leads
1. Direct-fetch r/ynab and r/MonarchMoney threads for verbatim quotes (not done — budget/access).
2. balancepro.app/blog/why-i-stopped-using-monarch-money — critical first-party post, worth fetching.
3. App Store / Google Play review pages identified but not individually read (ynab id1010865877; monarch id1459319842; play.google.com com.monarchmoney.mobile).
4. help.monarch.com blocks automated fetch (403) — relied on search snippets only.
5. Neither app's OCR/receipt-scanning support confirmed or denied by a primary source.
6. Notification/reminder feature sets not directly confirmed from primary docs for either app.

## Could not find
- Verbatim 1-3★ review quotes for either app.
- OCR/receipt-scanning confirmation either way.
- Independent (non-Monarch) numbers on Mint-to-Monarch migration share.
- Monarch's "13,000+ institutions" claim unconfirmed on an official page.

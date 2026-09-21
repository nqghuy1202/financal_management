# Digest: lightweight-trackers cluster (Spendee, Monefy) — round 1

Accessed date for all entries: 2026-09-21.

## SPENDEE — Q1 Feature teardown
- {claim: "AI Receipt Scanner (marked NEW) scans a receipt and auto-fills price/category/description/photo in seconds", source: mwm.ai/apps/expense-budget-app-spendee/635861140, publisher: MWM.ai (Spendee's publisher page), confidence: medium, class: official-doc}
- {claim: "Multiple Wallets to separate cash/accounts for trips/events/side projects; Multiple Currencies for travel", source: same mwm.ai page, confidence: medium, class: official-doc}
- {claim: "Shared Finances (split costs with partner/roommates/family); Labels to tag transactions for granular analysis", source: same page, confidence: medium, class: official-doc}
- {claim: "ML-based auto-categorization presented in visual graphs/insights ('personal financial advisor')", source: same page, confidence: medium, class: official-doc}
- {claim: "Supports bank-account syncing (not manual-only) — inferred from reviews referencing bank connection failing", source: trustpilot.com/review/spendee.com (reviews dated Jul/Sep 2025), confidence: medium (feature existence inferred from complaints), class: user-review}
- {claim: "Receipt scanning gated behind paid 'Plus' tier (~$2.49/mo)", source: appgrooves.com comparison pages (aggregated), confidence: low, class: aggregator}
- {claim: "Web version for desktop, dark mode, 'bank-level' secure sync", source: mwm.ai page, confidence: low (marketing language), class: official-doc}
- NOTE: could not directly fetch spendee.com or Play listing; feature claims rest on republished MWM.ai page + aggregators, not Spendee's own primary site.

## SPENDEE — Q2 Customer voice
- {claim: "5★: 'Easy and clear if you are looking for a simple expenses and budget tracking app'", source: trustpilot.com/review/spendee.com, date: Apr 2026, confidence: high, class: user-review}
- {claim: "1★ (mixed): liked exporting entries to Excel + categorizing, but 'can't connect to my bank' anymore", source: same Trustpilot, date: Jul 2025, confidence: high, class: user-review}
- {claim: "1★: 'the app has stopped synching which has rendered it useless' after years working", source: same Trustpilot, date: Sep 2025, confidence: high, class: user-review}
- {claim: "2★: 'Get worse and slower with every update'", source: same Trustpilot, date: Aug 2026, confidence: high, class: user-review}
- {claim: "1★: catastrophic data loss — 'all of a sudden my data was gone'", source: same Trustpilot, date: Jan 2026, confidence: high, class: user-review}
- {claim: "Auto-categorization ML 'isn't always dependable', sometimes miscategorizes", source: frugalforless.com/spendee-review + G2 (aggregated), confidence: medium, class: aggregator}
- {claim: "UI redesign removed home-page current-month view, replaced with just total balance — tracking-convenience regression", source: aggregated G2/App Store summary, confidence: low, class: aggregator}
- Pattern: praise = initial simplicity/clarity of categorizing; complaints overwhelmingly = bank-sync reliability breaking down over time, NOT manual-entry mechanics.

## MONEFY — Q1 Feature teardown
- {claim: "'Record any purchase in two taps'", source: monefy.com, publisher: Monefy official site, confidence: high, class: official-doc}
- {claim: "Entry flow: tap +/− then select category, no separate save step", source: aggregated App Store/review summaries, confidence: medium, class: aggregator}
- {claim: "Manual-only by design: 'import statements manually or add expenses in seconds, no bank login required'", source: monefy.com, confidence: high, class: official-doc}
- {claim: "150+ currencies supported", source: monefy.com, confidence: high, class: official-doc}
- {claim: "Paid tier unlocks unlimited accounts, recurring transactions, advanced filters; free tier gates these + custom categories + dark mode", source: monefy.com + aggregated app description, confidence: medium, class: official-doc/aggregator}
- {claim: "Sync via user's own Google Drive/Dropbox (not proprietary cloud), iOS+Android", source: monefy.com, confidence: high, class: official-doc}
- {claim: "Core UI = large interactive donut/pie chart for immediate category breakdown", source: aggregated app description, confidence: medium, class: official-doc}
- {claim: "No receipt scanning (per Spendee-vs-Monefy comparison)", source: appgrooves.com comparison (aggregated), confidence: low, class: aggregator}
- {claim: "Home-screen widgets for quick access", source: aggregated comparison/review pages, confidence: low, class: aggregator}
- {claim: "Passcode + biometric lock", source: monefy.com, confidence: high, class: official-doc}

## MONEFY — Q2 Customer voice
- {claim: "Praise: 'quick and easy logging of daily financial transactions'", source: justuseapp.com/en/app/1212024409/monefy-money-tracker/reviews, confidence: medium (aggregator, dates/stars unconfirmed), class: user-review}
- {claim: "Praise: 'I just wanted to be able to simply and manually enter expenses'", source: same justuseapp page, confidence: medium, class: user-review}
- {claim: "Praise: category granularity 'amazing feature', 'very useful' for tracking", source: same page, confidence: medium, class: user-review}
- {claim: "Complaint: no onboarding — user 'thought I had it but... Deleted my main account'", source: same page, confidence: medium, class: user-review}
- {claim: "Complaint: multi-device sync drift after erasing/replacing a device — 'never balanced correctly again'", source: same page, confidence: medium, class: user-review}
- {claim: "Felt gap: 'wish there was a monthly calendar so recurring events could be seen visually'; 'wish they had a way to easily track monthly subscriptions'", source: same page, confidence: medium, class: user-review}
- {claim: "UI limit: >14 categories don't all show icons on main screen; recurring options confusing in dark theme", source: same page, confidence: medium, class: user-review}
- {claim: "Broader themes (2nd search pass): instability/crashes, subscription-model/cancellation confusion, limited/unclear reporting customization", source: aggregated App Store review summaries, confidence: low, class: aggregator}
- {claim: "'Transactions added as notes, amount then category — quick but can become tedious if multiple missed'", source: aggregated (slant.co/incubatingwallet.com style), confidence: low, class: aggregator}
- {claim: "No bank sync at all frustrates busy/heavy users — framed as manual-only downside", source: aggregated, confidence: low, class: aggregator}
- Pattern: praise = raw entry speed (2-tap) + category visual tracking; complaints = no onboarding/accidental data loss, cross-device sync drift, paywalled convenience features, recurring-transactions lacking visual/calendar view.

## Cross-cutting takeaway (this assistant's own synthesis, flag as interpretation not raw claim)
- Bank-linking is double-edged: Spendee ships it but its #1 complaint is sync reliability breaking down — reliability/graceful-degradation matters more than the initial connect.
- Both apps' users prize FAST manual entry over automation — stronger/more corroborated signal than OCR/receipt-scan (weakly sourced, paywalled, not spontaneously praised in reviews found).
- Recurring-transaction visualization is a concrete, well-evidenced gap on Monefy (users want to see upcoming/missed subscriptions).
- Onboarding/first-use clarity is a concrete, well-evidenced Monefy pain point (accidental data loss).

## Leads
1. Fetch spendee.com and Monefy's Play listing directly (both failed/truncated this session).
2. Pull actual App Store/Play review pages w/ star+date metadata directly (only got Trustpilot for Spendee, JustUseApp aggregation for Monefy).
3. Check r/personalfinance/r/ynab or budgeting subreddits directly for unprompted Spendee/Monefy mentions.
4. Verify wearable support + true offline-entry-then-sync for both.
5. Verify Spendee widgets/recurring templates from Spendee's own docs (currently aggregator-only, low confidence).

## Could not find
- Spendee's own site content (spendee.com) — fetch attempts failed.
- Direct dated/starred verbatim Play Store reviews for either app.
- Wearable/Apple Watch support confirmation for either app.
- True offline-entry-with-later-sync confirmation for either app.
- Independent verification that Monefy lacks receipt scanning / Spendee's scanner is Plus-gated (single low-confidence aggregator source only).

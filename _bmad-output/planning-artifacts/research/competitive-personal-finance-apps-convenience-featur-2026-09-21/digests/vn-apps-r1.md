# Digest: VN apps cluster (Money Lover, MISA Sổ Thu Chi) — round 1

Accessed date for all entries: 2026-09-21.

## MONEY LOVER — Q1 Feature teardown
- {claim: "Siri integration + Apple Watch 'AI Voice' for hands-free expense logging", source: apps.apple.com/us/app/money-lover-s%E1%BB%95-thu-chi/id486312413?l=vi, publisher: Apple App Store, confidence: high, class: official-doc}
- {claim: "Receipt OCR auto-extracts transaction data from photos, incl. batch scanning of multiple receipts", source: same App Store listing, confidence: high, class: official-doc}
- {claim: "Multi-currency, built-in calculator, shared wallets, recurring-transaction automation, budget tracking/notifications, multi-device sync", source: same App Store listing, confidence: high, class: official-doc}
- {claim: "Bank linking with 25+ VN banks, bill-payment reminders, currency-exchange helper for trips, reports/charts", source: thegioididong.com/game-app/money-lover-... + moneylover.me/vi, confidence: medium (promotional tone, bank count unconfirmed independently), class: aggregator/official-doc}
- {claim: "Google Play listing emphasizes categorization + pie/bar chart insights, custom category budgets; free w/ paid premium tier", source: play.google.com/store/apps/details?id=com.tj.money.lover (search snippet, fetch truncated), confidence: medium, class: official-doc}
- {claim: "Lesser-known utilities: income-tax calculator, loan-interest calculator, custom icon packs, separate 'travel mode'", source: tinhte.vn/thread/chia-se-vai-tinh-nang-hay-ho-it-duoc-biet-den-tren-money-lover.3057453, publisher: Tinh Tế forum, confidence: medium, class: forum}

## MONEY LOVER — Q2 Customer voice
- {claim: "App Store 4.6/5, 2.3K ratings. 2★: no bank/card linking, no receipt scan, dislikes ads. 4★: 'useful' but 'very basic'. 5★: praises goal wallets for cash mgmt w/o card linking", source: same App Store listing, confidence: medium (summarized, not verbatim-pulled), class: user-review}
  - NOTE: 2★ complaint conflicts with the listing's own OCR/bank-link marketing — likely older version/region; flagged as inconsistency, not resolved.
- {claim: "Play reviews (via search summary): like daily tracking + cloud backup; complain recurring transactions (once free) now paywalled ~$35, features progressively paywalled", source: play.google.com listing via search snippet, confidence: low-medium, class: user-review/aggregator}
- {claim: "VOZ thread (403-blocked, secondhand summary only): 'poor UX, requires more taps than Misa', 'too many unnecessary features', 'weak debt/investment mgmt', 'app has gotten worse over time'", source: voz.vn/t/money-lover-thu-phi-cac-bac-co-nang-cap-len-khong.636227, confidence: low (unverified, could not directly read), class: forum}
- {claim: "Tinh Tế forum quotes: 'Quản lý cũng hay mà khâu ngồi nhập nhập làm biếng quá' (manual entry is tedious); 'Mình thấy giao diện kém xa misa' (UI worse than MISA); 'Mình cũng không liên kết ngân hàng' (reluctant to link bank, security); one praised it as faster than spreadsheets for small café bookkeeping", source: tinhte.vn thread above, confidence: medium (quotes surfaced via fetch, not raw-HTML-verified), class: user-review}

## MISA SỔ THU CHI (MoneyKeeper) — Q1 Feature teardown
- {claim: "Voice-memo recording of expenses, chat/message-style input, 'Scan hóa đơn: ghi chép không cần gõ' (scan receipts, no typing)", source: apps.apple.com/us/app/s%E1%BB%95-thu-chi-misa/id865818973?l=vi, publisher: Apple App Store, confidence: high, class: official-doc}
- {claim: "Bank-account linking w/ automatic import + AI categorization, recurring automation, multi-currency/multi-account (cash/bank/e-wallet/gold/FX/savings), shared/family accounts, budget alerts near limits, charts/reports, cross-device sync", source: same App Store listing, confidence: high, class: official-doc}
- {claim: "Official blog: transactions loggable in <1 min, 'quick record by scanning invoices', joint/shared-account for household mgmt", source: misa.vn/62264/so-thu-chi-money-keeper-tro-ly-tai-chinh-ca-nhan-mien-phi, publisher: MISA JSC, confidence: medium (self-published), class: official-doc/press}
- {claim: "Exchange-rate lookup + personal income-tax calculator add-ons", source: same misa.vn page, confidence: medium, class: official-doc/press}
- {claim: "Free Excel/PDF export, unlimited sharing for families/groups, multi-dimensional analysis reports (enterprise-accounting heritage)", source: premiumvns.com/so-sanh-so-thu-chi-misa-va-money-lover, publisher: 3rd-party comparison site (sells premium codes — commercial interest), confidence: low-medium, class: aggregator}

## MISA SỔ THU CHI — Q2 Customer voice
- {claim: "App Store 4.7/5, 846 ratings. Praise: 'easiest app... to keep track of bank accounts, credit cards and cash wallet'; 'Premium justified the cost after trying competitors'. Complaint: 'budget section does not sync' across devices while other features worked fine", source: apps.apple.com listing above, confidence: medium (summarized), class: user-review}
- {claim: "Search synthesis across VN sources: 'hasn't been updated in a long time, lacks dark mode, lacks collaboration features with other users'", source: aggregated incl. premiumvns.com comparison, confidence: low (search-synthesized, not individually verified), class: aggregator}
- {claim: "momo.vn blog (competitor's blog, commercial interest): MISA + MoMo are the only compared apps offering full basic expense mgmt free w/ no locked core features; Money Lover paywalls export/attachment/sync", source: momo.vn/blog/ung-dung-quan-ly-tai-chinh-ca-nhan-pho-bien-nhat-c103dt2282, confidence: low, class: aggregator/press}
- {claim: "Same MoMo blog: Money Lover's core weakness is fully-manual entry of every transaction, driving high drop-off for users without note-taking discipline", source: same momo.vn blog, confidence: low, class: aggregator/press}

## Leads
1. VOZ thread on Money Lover fees — 403-blocked, only secondhand summary; needs alternate fetch.
2. Tinh Tế thread "Review 1 số app quản lý thu chi cá nhân" (tinhte.vn/thread/.../3552556/page-3) — found, not fetched.
3. Google Play listings for both apps — JS-heavy, fetch truncated/failed; need dedicated scrape.
4. fptshop.com.vn review articles for both apps — found in search, not fetched.
5. No evidence gathered on: offline support, CSV/Excel *import* (only MISA export confirmed), widget support.

## Could not find
- Independent confirmation of Money Lover's "25+ VN banks" claim.
- Verbatim individually-dated 1-3★ reviews from Google Play for either app.
- CSV/Excel import capability (only MISA export documented) for either app.
- Offline-mode support for either app.
- Widget support for either app.

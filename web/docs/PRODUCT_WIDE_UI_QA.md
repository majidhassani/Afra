# Afra Product-Wide UI QA

This checklist tracks the July 2026 product-wide UI repair. Status is updated only after source review and browser validation.

| Route | Screen | Audit | Redesign | Responsive | RTL | Accessibility | Validation | Remaining risk |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `/login` | Sign in | In progress | In progress | Pending matrix | Pending | In progress | Build passed | Auth transition motion |
| `/register` | Registration | In progress | In progress | Pending matrix | Pending | In progress | Build passed | Long validation copy |
| `/app/dashboard` | HQ | Audited | In progress | 390 passed | Pending | Touch targets repairing | Mobile width passed | Desktop hierarchy |
| `/app/missions` | Mission discovery | Audited | In progress | 390 passed | Pending | Touch targets repairing | Mobile width passed | Card information density |
| `/app/missions/new` | Mission creation | Audited | In progress | 390 passed | Pending | Touch targets repairing | Mobile width passed | Segmented controls |
| `/app/missions/:id` | Mission command center | Audited | Complete | 390/430/768/1024/1440 passed | RTL/LTR passed | Passed | Build/typecheck/lint passed | Real production data may contain longer titles |
| `/app/missions/:id/map` | Mission map | Audited | In progress | 390 passed | Pending | In progress | Mobile width passed | Sheet drag/focus, real Maps key |
| `/app/missions/:id/locations/:id` | Location | Audited | In progress | 390 passed | Pending | In progress | Mobile width passed | Action confirmation |
| `/app/missions/:id/characters` | Character roster | Audited | In progress | 390 passed | Pending | In progress | Mobile width passed | Card hierarchy |
| `/app/missions/:id/characters/:id` | Character chat | Audited | In progress | 390 passed | Pending | In progress | Mobile width passed | Keyboard/composer |
| `/app/missions/:id/clues` | Evidence list | Audited | In progress | 390 passed | Pending | In progress | Mobile width passed | Clue/evidence distinction |
| `/app/missions/:id/clues/:id` | Evidence detail | Audited | In progress | 390 passed | Pending | In progress | Mobile width passed | Analysis hierarchy |
| `/app/missions/:id/ai` | Mission Control | Audited | In progress | 390 passed | Pending | Touch targets repairing | Mobile width passed | AI dominance balance |
| `/app/missions/:id/journal` | Journal | Audited | In progress | 390 passed | Pending | In progress | Mobile width passed | Delete confirmation |
| `/app/missions/:id/report` | Report center | Audited | In progress | 390 passed | Pending | Touch targets repairing | Mobile width passed | Irreversible action clarity |
| `/app/missions/:id/timeline` | Case timeline | Audited | In progress | 390 passed | Pending | In progress | Mobile width passed | Event hierarchy |
| `/app/missions/:id/events` | Raw event feed | Audited | In progress | 390 passed | Pending | In progress | Mobile width passed | Diagnostic positioning |
| `/app/missions/:id/time` | Time control | Audited | In progress | 390 passed | Pending | Touch targets repairing | Mobile width passed | Confirmation clarity |
| `/app/missions/:id/result` | Mission result | In progress | In progress | Pending matrix | Pending | In progress | Build passed | Mock completion state |
| `/app/profile` | Agent dossier | Audited | In progress | 390 passed | Pending | Touch targets repairing | Mobile width passed | Progression hierarchy |
| `/app/wallet` | Resources | Audited | In progress | 390 passed | Pending | In progress | Mobile width passed | Transaction states |
| `/app/history` | Case archive | Audited | In progress | 390 passed | Pending | In progress | Mobile width passed | Empty/completed variants |
| `/app/settings` | Settings | Audited | In progress | 390 passed | Pending | Touch targets repairing | Mobile width passed | Calm grouping |
| `/app/diagnostics` | Diagnostics | Audited | In progress | 390 passed | Pending | In progress | Mobile width passed | Dev-only route |

## Baseline findings

- No authenticated route produced document-level horizontal overflow at 390×844.
- Context navigation is fixed at five visible mobile destinations and does not scroll.
- Shared controls were commonly 31–42px tall; the canonical button, segmented control, guidance chip, onboarding control, portrait action, and timeline handle are being normalized to at least 44px.
- Multiple fixed overlay systems use independent z-index values; these require interaction checks together rather than isolated CSS review.
- The fallback map is testable in mock mode. Real Google Maps rendering remains an environment-dependent launch risk without a configured key.
- The mission summary card now uses natural height and visible overflow, with no internal scrollbar in either direction. Progress, risk, status, and metadata remain visible at all five required card viewports in English and Persian.

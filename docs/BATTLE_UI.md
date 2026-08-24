# Battle UI

The game view uses a full-viewport, table-first layout inspired by the visual hierarchy of *Marvel Duel*. The redesign keeps the existing game engine and choice flow intact while making the enemy area, player area, hand, and payment state readable at a glance.

## User-facing design

### Battlefield layout

- The board scales as one fixed scene to fit the browser viewport without page scrollbars.
- The villain command area occupies the upper battlefield; the active hero and player assets occupy the lower battlefield.
- The villain and main scheme form the upper visual axis, while the hero remains anchored to the lower-left command position.
- Main schemes use a suspended banner treatment. Heroes and villains use circular portrait medallions with health rings and name plates.
- Decorative metal corners, command rails, and battle dividers establish the table boundary without changing card hit areas.

### Card state and interaction

- HP, ATK, THW/SCH, threat, stage, counters, and status effects are rendered as high-contrast badges around the card.
- Exhausted cards remain visibly rotated or desaturated according to card type.
- The active player receives a gold focus glow.
- Cards in hand rise and highlight on hover while retaining the original click and multi-select payment behavior.
- The payment/question interface is a bottom glass drawer overlay. Opening it does not resize or push the battlefield.
- Invalid card codes (`undefined`, `null`, or empty) fall back to a safe card back instead of producing a broken image URL.

### Large card preview

Hover previews render through a `document.body` portal rather than inside the scaled board.

- Maximum preview height: approximately `72vh`.
- The preview is placed on the half of the screen opposite the hovered card.
- `position: fixed` and a high stacking layer prevent clipping by board overflow or transforms.
- `pointer-events: none` prevents hover feedback loops and preserves card selection.
- Coarse-pointer devices keep their existing tap/click interaction instead of opening a hover overlay.

### Character effects

Character effects are deliberately low-frequency and do not participate in layout:

- **Villain/Boss:** dark-red danger pulse, restrained glow, and moving scan-line texture.
- **Hero:** blue-gold breathing aura with a two-pixel floating motion.
- Effect layers never intercept pointer input.
- `prefers-reduced-motion` and the in-game animation switch disable these animations.

## Responsive and accessibility behavior

- Verified at desktop viewport sizes including `1920×1080` and `1366×768` without page scrollbars.
- The board camera performs uniform scaling, preserving card proportions and hit regions.
- Reduced-motion users receive static effects.
- Reduced-transparency users receive more opaque HUD surfaces.
- Functional state is not communicated through animation alone; badges and labels remain present when effects are disabled.

## Main implementation files

| File | Responsibility |
| --- | --- |
| `web/src/board/layout.ts` | Enemy, shared, player-asset, hero, pile, and hand positioning. |
| `web/src/components/Board.tsx` | Viewport fitting, battlefield scene, decorative rails, and card mounting. |
| `web/src/components/GameCard.tsx` | Card rendering, portrait medallions, badges, safe-code fallback, and character effect layers. |
| `web/src/cards.tsx` | Portal-based large card preview and image fallback behavior. |
| `web/src/views/Game.tsx` | Choice navigation, card selection, payment confirmation, HUD, and reconnect-safe view updates. |
| `web/src/style/board.css` | Battlefield theme, card state, hand motion, HUD drawer, character effects, and responsive rules. |
| `web/src/style.css` | Global portal preview sizing and placement. |

## Interaction guarantees

The visual redesign must not change these gameplay contracts:

1. Board cards and QuestionPanel choices resolve through the same choice IDs.
2. `choose_n` resource payment remains a true multi-select operation.
3. Preview overlays never consume clicks or pointer events.
4. Opening the payment drawer never changes board dimensions.
5. WebSocket reconnect and persisted game snapshots remain independent of presentation state.
6. Animation-disabled mode remains fully playable.

## Validation

For UI changes, run:

```bash
cd web
npm run build
```

Recommended manual checks:

1. Open a live game at `1920×1080` and `1366×768`.
2. Confirm there are no document scrollbars.
3. Hover cards on both sides of the board and verify the preview appears opposite the anchor.
4. Select several resource cards and ensure the payment confirmation remains visible and clickable.
5. Toggle animation off and enable OS reduced-motion mode.
6. Verify hero/villain cards remain clickable while effects are active.

## Change history

- `5bac267` — full-viewport Marvel Duel-style battlefield redesign.
- `ca8d2b1` — viewport-scale card previews and hero/villain character effects.

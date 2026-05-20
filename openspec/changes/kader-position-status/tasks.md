## 1. Create PositionStatus Component

- [ ] 1.1 Create `web/src/components/PositionStatus.tsx` with position constants (TW, LA, RA, RL, RM, RR, KL)
- [ ] 1.2 Implement position count logic: aggregate positions from member array
- [ ] 1.3 Implement color mapping: red (0), yellow (1), green (2), blue (3+)
- [ ] 1.4 Render positions horizontally with vertically stacked circles per position

## 2. Integrate into AdminKaderPage

- [ ] 2.1 Import PositionStatus component in AdminKaderPage.tsx
- [ ] 2.2 Add `<PositionStatus members={k.members} />` between mode toggle and trainer search
- [ ] 2.3 Verify positioning matches design (between Jahrgänge and Trainer sections)

## 3. Styling & Polish

- [ ] 3.1 Set circle diameter to 14px (use Tailwind or inline size)
- [ ] 3.2 Add appropriate spacing between positions horizontally (8-12px gap)
- [ ] 3.3 Add appropriate spacing between vertically stacked circles (2-4px gap)
- [ ] 3.4 Ensure position abbreviations are same font size as Jahrgänge badge
- [ ] 3.5 Center circles vertically within each position column
- [ ] 3.6 Verify layout is compact and doesn't wrap on desktop/tablet
- [ ] 3.7 Test on mobile: circles remain visible and don't break layout

## 4. Testing

- [ ] 4.1 Test with Kader having 0 members in a position (should show 1 red circle)
- [ ] 4.2 Test with Kader having 1 member in a position (should show 1 yellow circle)
- [ ] 4.3 Test with Kader having 2 members in a position (should show 2 green circles stacked vertically)
- [ ] 4.4 Test with Kader having 3+ members in a position (should show 3 blue circles stacked vertically)
- [ ] 4.5 Test with members having multiple positions (count correctly per position)
- [ ] 4.6 Test empty Kader (all positions should show 1 red circle each)
- [ ] 4.7 Verify positions remain horizontally aligned and circles don't wrap
- [ ] 4.8 Verify no console errors or accessibility warnings

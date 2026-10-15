# Look v3 visual evidence

`final/` contains cache-disabled full-page screenshots of the real design preview at
375×900 and 1440×900, before and after selecting Belém Tower, plus measured component bounds.
Local place images are verified by HTTP and browser tests. SwiftShader supplies WebGL.

Each `V1/`–`V12/` directory contains the corresponding implementation crop and a comparison:
**implementation on the left, mockup crop on the right**. V12 is phone-only. The phone mockup
is cropped inside its device frame; the implementation includes the complete scrolling page.
The phone reference does not include the writers card, stays or videos below the visible crop;
V10 therefore uses the desktop comparison. Extra phone teasers implement decision D9.

- [Desktop overview](final/1440-city.png)
- [Desktop selected place](final/1440-place.png)
- [Phone overview](final/375-city.png)
- [Phone selected place](final/375-place.png)
- [Desktop comparison](V6/1440-comparison.png)
- [Phone comparison](V12/375-comparison.png)

The offline fixture uses a blank local map style and synthetic category photography. The
reference's roads, labels and real place photography are consequently absent. Selecting a
place moves the camera closer; the city overview shows the dispersed fixture pins. Weather,
writer post titles and stay/video availability use existing data, with no invented price,
excerpt, duration or extra photos. These are layout/runtime comparisons, not evidence of
production tile availability or pixel-identical photography.

`V6/*-map.png` and `V8/*-panel.png` preserve intermediate screenshot evidence; the
`*-implementation.png` / `*-comparison.png` files and `final/` contain the final captures.
`V13/` preserves the Funchal search/save/plan regression journey. `a11y/` contains the final
accessibility audit's machine-readable results and screenshots.

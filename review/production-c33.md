# C33: Logic analyzer interaction and Phi2

- Manual scroll/wheel/pointer interaction disables automatic following without stopping capture.
- FOLLOW explicitly resumes right-edge scrolling; LIVE returns to the newest page and enables following.
- Table-row selection centers the matching graph cycle, and graph selection brings its table row into view.
- Cycle indices appear along the horizontal waveform axis.
- Phi2 draws a complete ideal 50% duty-cycle waveform per CPU bus cycle, with the low and high half-cycles visibly alternating rather than a constant sampled high level.
- Analyzer JavaScript syntax and Go tests validated.

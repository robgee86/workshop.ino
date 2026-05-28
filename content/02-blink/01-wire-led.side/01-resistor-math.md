---
title: Resistor Math
---

An LED needs current-limiting or it burns out. Ohm's law gives the resistor value:

```
R = (Vsupply − Vled) / Iled
  = (5V − 2V) / 0.015A
  = 200Ω  →  round up to 220Ω
```

Pick the next standard value **above** the calculated one to stay safe. A 330Ω
resistor works too — the LED is just a little dimmer.

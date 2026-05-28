---
title: Wire the LED
summary: Build the circuit on a breadboard.
attachments:
  - path: ./blink-starter.ino
    label: Starter sketch
    description: The empty sketch you'll fill in next.
  - path: ../assets/wiring.svg
    label: Wiring diagram (SVG)
    description: Printable version of the circuit below.
links:
  - url: https://docs.arduino.cc/learn/microcontrollers/digital-pins/
    label: Digital pins — Arduino docs
    description: How HIGH/LOW, inputs and outputs work on Arduino pins.
  - url: https://en.wikipedia.org/wiki/Light-emitting_diode
    label: Light-emitting diode — Wikipedia
    description: Why LEDs need a current-limiting resistor (and how much).
---

Build this circuit: pin **D13** → LED (long leg) → **220Ω resistor** → **GND**.

![LED wiring diagram](../assets/wiring.svg)

The current path, in order:

```mermaid
flowchart LR
  D13 --> LED --> R[220Ω] --> GND
```

We declare which pin drives the LED. In the next step we'll make it blink:

```c
const int LED_PIN = 13;   // built-in LED
```

> ◇ Curious why the resistor matters, or how to drive an RGB LED? Check the side
> quests below before moving on.

**Checkpoint:** your circuit matches the diagram.

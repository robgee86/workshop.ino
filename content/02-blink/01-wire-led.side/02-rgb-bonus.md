---
title: RGB Bonus
---

Want colour? An RGB LED is just three LEDs in one package. Wire each leg to its
own pin through its own resistor, then mix:

```c
void setColor(int r, int g, int b) {
  analogWrite(9,  r);  // red
  analogWrite(10, g);  // green
  analogWrite(11, b);  // blue
}
```

Call `setColor(255, 0, 128)` for a pink glow. Use **PWM-capable** pins (marked `~`).

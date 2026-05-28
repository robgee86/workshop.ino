---
title: Connect Your Board
summary: Plug in and select the right port.
---

Connect the Arduino to your laptop with a USB cable. The IDE talks to the board
through a serial port, which you must select.

```mermaid
flowchart LR
  A[Arduino board] -- USB --> B[Your laptop]
  B --> C{IDE: select port}
  C -->|correct port| D[Ready to upload]
  C -->|wrong port| E[Upload fails]
```

In the IDE, choose **Tools → Board** (your model) and **Tools → Port** (the one that
appears when you plug the board in).

**Checkpoint:** the board model and a port are both selected.

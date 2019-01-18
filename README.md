# dedinamik
## Lightweight, power saving proxy

![stability-wip](https://img.shields.io/badge/stability-wip-red.svg)

----
### A lightweight proxy ?
Dedinamik acts like a proxy with TCP/UDP connections.
It is able to forward requests from an open port to another one.

### Power saving ?
Dedinamik aims to help low end servers to handle a lot of different apps by "freezing" unused ones.
Two different power saving modes are currently handled.
The "freezing" option just freeze unused apps to stop CPU consumption.
The "non freezing" option means that the unused apps will be completely stopped then restarted on demand. It's the best to keep your server clean of consumption, but it is only useable correctly with fast startup apps.

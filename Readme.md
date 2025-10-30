# Sunrise

A monitor-restart service written for Sunshine Linux Hosts!

## What does this do?

The great [Sunshine](https://app.lizardbyte.dev/Sunshine/) game stream
application has a critical issue on Linux desktops: When the display sleeps,
Sunshine doesn't wake it up and instead errors out (see [this GiHub discussion
for more information](https://github.com/orgs/LizardByte/discussions/439)). This
makes using Sunshine with Linux-on-the-Desktop a frustrating experience.

Sunrise will monitor the Sunshine log file, watch for monitor-is-missing errors,
then wake the monitor.

Sunrise is very configurable, check out the `sunrise.cfg.example` file for all
of the commented options.

## How do I use it?

* Build sunrise with `go build` (or use `build-with-docker.bash` to build using
  an ephemeral Docker container)
* Move `sunrise` to `/opt/sunrise/sunrise`
* Copy `sunrise.cfg.example` to `/etc/sunrise/sunrise.cfg`
* Edit `/etc/sunrise/sunrise.cfg` to your specifications (you'll need to change
  the `SunshineLogPath` variable)
* Copy `sunrise.service` to `$HOME/.config/systemd/user/sunrise.service`
* `systemctl --user daemon-reload`
* `systemctl --user enable sunrise`
* `systemctl --user start sunrise`

## How can I help?

This is working for Debian Trixie + KDE and GNOME Wayland, but I don't have the
time to test _every_ desktop environment out there. If you want, feel free to
add other working "wake monitor" commands in `sunrise.cfg.example`.

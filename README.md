## TSP (TUI Sigmo's Power Manager) is a power manager built on top of TLP and Powertop

Features:
  - Displays total power consumption (W), device temperature, CPU frequency, and CPU load
  - Manages TLP power profiles (power-saver, balanced, performance, ac)
  - Toggles CPU Turbo Boost on/off
  - Runs `powertop --auto-tune` to optimize tunables

## ⚠️ Warning

This program is in active development, use at your own risk.

**The install script currently only supports Arch Linux and its derivatives — do not run it on other distributions!**

The install script `install.sh` makes the following changes to your system:
  - Installs `tlp`, `tlp-rdw`, `powertop`, `opendoas`, and `go`
  - Removes `power-profiles-daemon` if found, after asking for user confirmation
  - Overwrites `/etc/tlp.conf` — if you have your own config, make sure to back it up first
  - Adds `permit nopass <you> as root` to `/etc/doas.conf`. This means after installation, you can run commands as root without a password via `doas`, which is exactly what the program does when invoking TLP and Powertop commands

-----------------------------------------------------------------------

Planned:
 - Add an option to run `tlp fullcharge`
 - Ability to configure TLP settings directly from the TUI
 - Add remaining battery time display 
 - Improve the user interface
 - Add configuration quality assessment based on all collected metrics

<img width="789" height="604" alt="image" src="https://github.com/user-attachments/assets/39902a72-5e05-4ada-bfbb-0edb05a429ae" /> 

## Installation via Script

```bash
git clone https://github.com/trashmania/tsp.git
cd tsp
chmod +x install.sh
./install.sh

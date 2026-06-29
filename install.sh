#!/usr/bin/env bash
set -e

echo "=== Installing the best power manager as of 2026 ==="
echo "=== TSP [Thinkpad Sigmo's Power manager] ==="
echo "------------------------------------------------------------"

if pacman -Qq power-profiles-daemon &>/dev/null; then
    echo "[!] Detected power-profiles-daemon. It conflicts with TLP and TSP."
    read -p "[?] Remove power-profiles-daemon? (y/n): " choice
    if [[ "$choice" == "y" || "$choice" == "Y" ]]; then
        sudo pacman -Rns power-profiles-daemon --noconfirm
    else
        echo "[-] Warning: TSP stability is not guaranteed without removing ppd!"
    fi
fi

echo "[+] Installing required packages..."
sudo pacman -S opendoas tlp tlp-rdw powertop go --needed --noconfirm

if [ ! -f /etc/doas.conf ] || ! grep -q "permit nopass $USER" /etc/doas.conf; then
    echo "[+] Configuring /etc/doas.conf for the current user ($USER)..."
    echo "permit nopass $USER as root" | sudo tee -a /etc/doas.conf > /dev/null
    sudo chmod 600 /etc/doas.conf
fi

if [ -f "tlp.conf" ]; then
    echo "[+] Applying default TLP configuration..."
    sudo cp tlp.conf /etc/tlp.conf
fi

echo "[+] Activating TLP system service..."
sudo systemctl enable --now tlp.service

echo "[+] Building TSP binary..."
go build -ldflags="-s -w" -o tsp main.go

echo "[+] Copying TSP to /usr/local/bin/..."
doas cp tsp /usr/local/bin/

echo "------------------------------------------------------------"
echo "[+] Installation complete! You can call the program by typing 'tsp' in your terminal."
echo "[+] Enjoy!"

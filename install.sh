#!/usr/bin/env bash
set -e

echo "=== Установка лучшего на момент 2026 года менеджера питания ==="
echo "=== TSP [Thinkpad Sigmo's Power manager] ==="
echo "------------------------------------------------------------"

if pacman -Qq power-profiles-daemon &>/dev/null; then
    echo "[!] Обнаружен power-profiles-daemon. Он конфликтует с TLP и TSP."
    read -p "[?] Удалить power-profiles-daemon? (y/n): " choice
    if [[ "$choice" == "y" || "$choice" == "Y" ]]; then
        sudo pacman -Rns power-profiles-daemon --noconfirm
    else
        echo "[-] Внимание: без удаления ppd стабильная работа TSP не гарантируется!"
    fi
fi

echo "[+] Установка необходимых пакетов..."
sudo pacman -S opendoas tlp tlp-rdw powertop go --needed --noconfirm

if [ ! -f /etc/doas.conf ] || ! grep -q "permit nopass $USER" /etc/doas.conf; then
    echo "[+] Настройка /etc/doas.conf для текущего пользователя ($USER)..."
    echo "permit nopass $USER as root" | sudo tee -a /etc/doas.conf > /dev/null
    sudo chmod 600 /etc/doas.conf
fi

if [ -f "tlp.conf" ]; then
    echo "[+] Применение дефолтной конфигурации TLP..."
    sudo cp tlp.conf /etc/tlp.conf
fi

echo "[+] Активация системной службы TLP..."
sudo systemctl enable --now tlp.service

echo "[+] Сборка бинарника TSP..."
go build -ldflags="-s -w" -o tsp main.go

echo "[+] Копирование TSP в /usr/local/bin/..."
doas cp tsp /usr/local/bin/

echo "------------------------------------------------------------"
echo "[+] Конец установки! Для вызова программы напишите 'tsp' в терминале."
echo "[+] Приятного использования!"

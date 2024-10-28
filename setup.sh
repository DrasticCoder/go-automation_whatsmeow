#!/bin/bash

# Check if the script is being run on Linux or Windows
OS="unknown"
case "$(uname -s)" in
    Linux*)     OS="linux";;
    Darwin*)    OS="macos";; # Optional support for macOS if needed
    CYGWIN*|MINGW*|MSYS*) OS="windows";;
    *)          OS="unknown";;
esac

echo "Detected OS: $OS"

# Function to install Go on Linux
install_go_linux() {
    echo "Checking for Go installation..."
    if ! command -v go &> /dev/null; then
        echo "Go not found, installing..."
        wget https://golang.org/dl/go1.19.3.linux-amd64.tar.gz
        sudo tar -C /usr/local -xzf go1.19.3.linux-amd64.tar.gz
        export PATH=$PATH:/usr/local/go/bin
        echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.profile
        source ~/.profile
    else
        echo "Go is already installed."
    fi
}

# Function to install Go on Windows
install_go_windows() {
    echo "Checking for Go installation..."
    if ! command -v go &> /dev/null; then
        echo "Go not found, downloading..."
        curl -LO https://golang.org/dl/go1.19.3.windows-amd64.msi
        echo "Installing Go..."
        msiexec /i go1.19.3.windows-amd64.msi /quiet
    else
        echo "Go is already installed."
    fi
}

# Function to install GCC on Linux
install_gcc_linux() {
    echo "Checking for GCC installation..."
    if ! command -v gcc &> /dev/null; then
        echo "GCC not found, installing..."
        sudo apt-get update
        sudo apt-get install -y build-essential
    else
        echo "GCC is already installed."
    fi
}

# Function to install GCC on Windows using Chocolatey
install_gcc_windows() {
    echo "Checking for GCC installation..."
    if ! command -v gcc &> /dev/null; then
        echo "GCC not found, installing..."
        if ! command -v choco &> /dev/null; then
            echo "Chocolatey not found, installing..."
            powershell -Command "Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://chocolatey.org/install.ps1'))"
        fi
        echo "Installing GCC via Chocolatey..."
        choco install -y mingw
    else
        echo "GCC is already installed."
    fi
}

# Main setup script
setup() {
    if [ "$OS" = "linux" ]; then
        install_go_linux
        install_gcc_linux
    elif [ "$OS" = "windows" ]; then
        install_go_windows
        install_gcc_windows
    else
        echo "Unsupported OS or environment."
        exit 1
    fi

    echo "Building Go executable..."

    # Build the Go application
    if [ "$OS" = "windows" ]; then
        GOOS=windows GOARCH=amd64 go build -o WhatsAppBuddy.exe main.go
        if [ $? -eq 0 ]; then
            echo "Build successful: WhatsAppBuddy.exe created."
            echo "Starting the server..."
            ./WhatsAppBuddy.exe &
        else
            echo "Build failed."
            exit 1
        fi
    else
        go build -o WhatsAppBuddy main.go
        if [ $? -eq 0 ]; then
            echo "Build successful: WhatsAppBuddy created."
            echo "Starting the server..."
            ./WhatsAppBuddy &
        else
            echo "Build failed."
            exit 1
        fi
    fi
}

# Execute the setup function
setup

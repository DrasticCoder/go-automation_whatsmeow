
# WhatsAppBuddy

WhatsAppBuddy is a powerful and secure WhatsApp bot built using GoLang and the WhatsMeow library. It provides businesses with essential tools for managing WhatsApp communications efficiently. Unlike other bots that rely on web scraping, WhatsAppBuddy offers a more secure and faster solution by leveraging official APIs and preserving WhatsApp sessions.

## Features

- **Send Bulk Messages**: Effortlessly send messages to multiple recipients with a single command, saving time and improving communication efficiency.
- **View Analytics**: Gain insights into message delivery, engagement rates, and customer interactions with comprehensive analytics.
- **Monitor Incoming Messages**: Keep track of incoming messages in real-time to ensure prompt responses to customer queries.
- **Secure and Fast**: Built with GoLang and WhatsMeow, WhatsAppBuddy offers enhanced security and performance compared to bots using web scraping tools. It provides a reliable and efficient communication solution without relying on the default WhatsApp interface.

## Installation

### Prerequisites

- [Go](https://golang.org/doc/install) (version 1.17 or higher)
- [SQLite3](https://www.sqlite.org/download.html) for the database

### Getting Started

1. **Clone the Repository**:
   ```sh
   git clone https://github.com/drasticcoder/whatsappBuddy.git
   cd whatsappBuddy
   ```

2. **Install Dependencies**:
   ```sh
   go mod download
   ```

3. **Build the Application**:
   ```sh
   go build -o whatsappBuddy
   ```

4. **Run the Application**:
   ```sh
   ./whatsappBuddy
   ```

   Ensure that you have configured your environment and database settings properly before running the application.

## Configuration

- **Database**: The application uses SQLite3 for data storage. Ensure that the SQLite3 database file is correctly specified in the application’s configuration.

- **WhatsApp Session**: The first time you run the application, you’ll need to scan the QR code to authenticate your WhatsApp account. Follow the on-screen instructions to complete the authentication process.

## Usage

- **Access the Application**: Open your web browser and navigate to `http://localhost:8080` to access the WhatsAppBuddy interface.
  
- **Send Bulk Messages**: Use the `/upload` endpoint to upload a CSV file containing phone numbers and a message to send bulk messages.

- **View Analytics**: Navigate to the `/analytics` page to view detailed analytics about message delivery and engagement.

- **Monitor Incoming Messages**: Access the `/messages` page to view real-time incoming messages.

## Security

WhatsAppBuddy prioritizes security by avoiding web scraping techniques, which can be prone to security risks and violations of WhatsApp’s terms of service. Instead, it uses official APIs and maintains a preserved session to ensure a secure and reliable experience.


# PricePulse 📈

PricePulse is a backend service designed to help cryptocurrency investors monitor market movements. It fetches real-time price data and checks it against user-defined signals to detect significant price changes.

---

### Core Features

- **RESTful API**: Endpoints to manage users and price-change signals.
- **Real-Time Data**: Fetches live cryptocurrency prices from the CoinGecko API.
- **Persistent Storage**: Uses Google Firestore to store user data, signals, and price history.
- **Signal Evaluation**: Contains business logic to detect when a price moves beyond a user's percentage-based threshold.
- **Tested**: Includes a suite of unit and integration tests.
- **Containerized**: A Dockerfile is included for building and deploying in a production environment like Google Cloud Run.
- **Health Monitoring**: A `/health` endpoint for uptime monitoring.

---

### Getting Started & Running Locally

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

#### Prerequisites

Before you can run this project locally, ensure you have the following tools installed:

- **Go**: The programming language used for this application. [Download Go](https://golang.org/dl/)
- **Google Cloud SDK (gcloud)**: Required to run the local Firestore emulator. [Installation Guide](https://cloud.google.com/sdk/docs/install)
- **Docker Desktop** (Optional): Useful for building containers to deploy the application to services like Google Cloud Run. [Download Docker](https://www.docker.com/products/docker-desktop/)

### Running Locally

Running the application locally requires two terminal windows: one for the database emulator and one for the Go application itself.

#### 1. Start the Firestore Emulator

In your first terminal window, start the local Firestore emulator. This terminal must remain open while you work.

```bash
gcloud emulators firestore start --host-port="localhost:8081"
```

#### 2. Run the Go Application

In a second terminal window, navigate to the project directory and run the application. Set the `FIRESTORE_EMULATOR_HOST` environment variable so the app connects to the local emulator instead of the live cloud database.

On macOS or Linux:

```bash
FIRESTORE_EMULATOR_HOST="localhost:8081" go run .
```

On Windows (Command Prompt):

```cmd
set FIRESTORE_EMULATOR_HOST=localhost:8081
go run .
```

You should see a log message indicating the server has started on port `8080`. You can now interact with the API using tools like `curl` or another API client.

---

### Running Tests

The integration tests also require the Firestore emulator to be running.

#### 1. Ensure the Emulator is Running

Make sure your first terminal window (from the "Running Locally" steps) is still running the emulator.

#### 2. Run the Test Suite

In a new terminal, navigate to the project directory and run the test command, again providing the environment variable.

On macOS or Linux:

```bash
FIRESTORE_EMULATOR_HOST="localhost:8081" go test -v ./...
```

On Windows (Command Prompt):

```cmd
set FIRESTORE_EMULATOR_HOST=localhost:8081
go test -v ./...
```

The `-v` flag provides verbose output, and `./...` ensures tests in all subdirectories are executed.

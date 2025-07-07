### Mango-Go
This project is a rewrite of the self-hosted manga server and web reader, [Mango](https://github.com/vrsandeep/Mango/), from Crystal into the Go programming language. The goal is to create a modern, performant, and easy-to-maintain application that retains the core functionality of the original while leveraging the strengths of Go's ecosystem.


## Installation & Usage

There are two primary ways to run Mango-Go: via Docker (recommended for production) or as a single, standalone binary.

### Option 1: Production Deployment (Docker)

The recommended way to run Mango-Go in production is by using Docker and Docker Compose. This ensures a consistent, secure, and easily manageable setup.

### Prerequisites

* **Docker**: [Install Docker](https://docs.docker.com/get-docker/)
* **Docker Compose**: [Install Docker Compose](https://docs.docker.com/compose/install/)

### Quick Start

1.  **Clone the Repository:**
    ```sh
    git clone https://github.com/vrsandeep/mango-go.git
    cd mango-go
    ```

2.  **Configure Your Library:**
    Open the `docker-compose.yml` file and find the `volumes` section. Change the line:
    ```yml
    - ./manga:/manga
    ```
    to point to the actual location of your manga library on your computer. For example:
    ```yml
    - /home/user/comics:/manga
    ```

3.  **Start the Application:**
    Run the following command from the root of the project directory:
    ```sh
    docker-compose up -d
    ```
    This will build the Docker image and start the Mango-Go container in the background.

4.  **First Run (Admin User):**
    The first time you start the application, it will create a default `admin` user. Check the container logs to get the randomly generated password:
    ```sh
    docker-compose logs
    ```
    Look for a message like:
    ```
    Default admin user created.
    Username: admin
    Password: <randomly_generated_password>
    ```

5.  **Access Mango-Go:**
    Open your web browser and navigate to `http://localhost:8080`. Log in with the admin credentials. It is highly recommended to change this password immediately via the Admin > User Management page.

### Option 2: Standalone Binary (Local/Manual Deployment)

The application can be compiled into a single, portable binary file that contains all necessary web assets and migrations. This is great for quick local use or for deploying on systems without Docker.

1.  **Build the Binary:**
    From the root of the project directory, run the following command:
    ```sh
    go build -o mango-go .
    ```
    This will create an executable file named `mango-go` (or `mango-go.exe` on Windows).

2.  **Configure:**
    Create a `config.yml` file in the same directory as the binary. You can also use environment variables (see `docker-compose.yml`).
    ```yml
    # config.yml
    library:
      path: "/path/to/your/manga"
    database:
      path: "./mango.db"
    port: 8080
    ```

3.  **Run the Application:**
    Simply execute the binary:
    ```sh
    ./mango-go
    ```
    The application will start, and you can access it at `http://localhost:8080`. All data, including the database, will be stored in the same directory.


### Data Persistence

The `docker-compose.yml` file is configured to store all application data (the SQLite database, config, etc.) in a `./data` directory on your host machine. Your manga library is mounted directly into the container and is never modified. This ensures that your data is safe even if you update or restart the container.

### Configuration

While a `config.yml` file exists, it is best practice to configure the application using **environment variables** in the `docker-compose.yml` file. This allows for more flexible deployments. See the `environment` section in the file for examples.


### Development

#### Project Structure
The project follows a standard Go layout:

- `/cmd/mango-cli`: The entry point for the command-line application.
- `/internal`: Contains all the core application logic, separated by concern:
- `/config`: Configuration loading.
- `/db`: Database initialization.
- `/library`: The main library scanning and parsing logic.
- `/models`: Core data structures.
- `/store`: The data access layer for all database queries.
- `/migrations`: SQL files for database schema migrations.

#### Running Tests
To run all unit and integration tests, execute the following command from the project root:

```sh
go test ./...
```

The tests use an in-memory SQLite database to ensure they run quickly and do not interfere with your main mango.db file.

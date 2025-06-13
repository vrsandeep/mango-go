### Mango-Go
This project is a rewrite of the self-hosted manga server and web reader, [Mango](https://github.com/vrsandeep/Mango/), from Crystal into the Go programming language. The goal is to create a modern, performant, and easy-to-maintain application that retains the core functionality of the original while leveraging the strengths of Go's ecosystem.

### Configuration

1. Create a config.yml file in the root of the project directory. You can use the provided template seen in [config.yml][./config.yml]
2. **Edit the config file**:
- Change `library.path` to the absolute or relative path of your manga collection. For testing, you can create a `./manga` directory.
- Change where the database should reside using `database.path`. It will be created at the specified location.

### Usage
Run the command-line scanner from the root of the project:

```sh
go run ./cmd/mango-cli/
```

This command will:

1. Read your `config.yml`.

2. Connect to the SQLite database and create it if it doesn't exist.

3. Run the necessary database migrations to set up the tables.

4. Scan the directory specified in `library.path`.

5. Populate the `mango.db` file with your library's metadata.

6. You should see log output indicating the progress of the scan.


#### Docker

The container can be run with:
`docker run -v /path/to/your/manga:/manga vrsandeep/mango-go`
To build the Docker image, run:
`docker build -t mango-go .`
To run the Docker container, use:
`docker run -d -v /path/to/your/manga:/manga vrsandeep/mango-go`

To run the container with a specific configuration file and a persistent database:
`docker run -v /path/to/your/manga:/manga -v /path/to/your/config.yml:./config.yml -v /path/to/db:./mango.db vrsandeep/mango-go`


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

#### Roadmap
TBD
# Backend
This directory contains the code for running the Go backend for Trackerr.
The purpose of the backend is to communicate with GPS Trackers using GT06 or JT808, store the relevant data in a database, and then provide a REST API, allowing for easy integration with other systems

## Prerequisites
Ensure you have the following installed:
- Go(version 1.24 or higher)
- Make
- Sqlite3

## Installation
Clone the repository
```
git clone https://github.com/i74578/Trackerr
cd Trackerr/backend
```
Install the required dependencies
```
make
```

## Configuration
Configurations are set in .env
SERVER_IP, must be set to the public IP address of the server. This is only used for the provisioning trackers, since they must be provided with a IP address to connect to.
TACKERCOM_PORT, refers to the tcp port listening for tracker communication
API_PORT, refers to the port used by the API

## Usage
This program leverages a makefile with several useful commands to simplify common operations
### Building the Project
```
make build
```
This will create an execuatable in the ./bin directory
## Running the Program
To run the program:
```
make
```
## Generating OpenAPI Specifications
To generate the OpenAPI speciications:
```
make generate-spec
```
This will generate specification files in the ../docs directory
## Create, Read, Update and Delete users
User management is **NOT** provided in the REST API, and must therefore be done manually. The following contains the commands needed to perform the CRUD operations.
### Create user
```
sqlite3 ./database.db \
  "INSERT INTO users ('name','apikey','admin','enabled') VALUES ('<name>',<apikey>,<admin 0|1>,<enabled 0|1>)"
```

### Read users
```
sqlite3 ./database.db \
  -header \
  -column \
  "SELECT * FROM users"
```

### Update user
```
sqlite3 ./database.db \
  "UPDATE users SET <property>=<value> WHERE id=<id>"
```

### Delete user
```
sqlite3 ./database.db \
  "DELETE FROM users WHERE id=<id>;"
```

🚨 **Please note:** It’s **strongly recommended** that you change the default admin api key before running in production.  

## Test REST API using Swagger
1. Start up a docker container running Swagger and bind the OpenAPI spec directory to it.
```
docker run --rm -p 8081:8080 \
  -e SWAGGER_JSON=/spec/swagger.yaml \
  -v "$(pwd)/../docs/swagger.yaml":/spec/swagger.yaml \
  swaggerapi/swagger-ui
```
2. Open [http://127.0.0.1:8081/](http://127.0.0.1:8081/) in a browser
3. Click on the authorize button and enter the API key, which by default is **AAAAAA**
